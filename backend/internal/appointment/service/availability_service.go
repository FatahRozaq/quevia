package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FatahRozaq/quevia/backend/internal/platform/database"
	"github.com/FatahRozaq/quevia/backend/internal/platform/database/repository"
	"github.com/FatahRozaq/quevia/backend/internal/repository/payload"
)

func (s *AppointmentService) CalculateAvailability(ctx context.Context, tenantID int64, branchGUID, serviceGUID uuid.UUID, dateStr string, staffGUID *uuid.UUID) (*payload.AvailabilityResponse, error) {
	var result *payload.AvailabilityResponse

	err := database.WithTx(ctx, s.db, &tenantID, func(tx pgx.Tx, q *repository.Queries) error {
		branch, err := q.GetBranchByGUID(ctx, pgtype.UUID{Bytes: branchGUID, Valid: true})
		if err != nil {
			return ErrBranchNotFound
		}

		svc, err := q.GetServiceByGUID(ctx, pgtype.UUID{Bytes: serviceGUID, Valid: true})
		if err != nil {
			return ErrServiceNotFound
		}

		loc, err := time.LoadLocation(branch.Timezone)
		if err != nil {
			loc = time.FixedZone("UTC+7", 7*3600)
		}

		targetDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			return fmt.Errorf("parsing date: %w", err)
		}

		dayOfWeek := int16(targetDate.Weekday())
		if dayOfWeek == 0 {
			dayOfWeek = 7
		}

		// 1. Check branch business hours
		bHours, err := q.ListBusinessHoursByBranch(ctx, branch.ID)
		if err != nil {
			return err
		}

		var openMicro, closeMicro int64
		hasBusinessHours := false
		for _, bh := range bHours {
			if bh.DayOfWeek == dayOfWeek {
				openMicro = bh.OpenTime.Microseconds
				closeMicro = bh.CloseTime.Microseconds
				hasBusinessHours = true
				break
			}
		}

		if !hasBusinessHours {
			result = &payload.AvailabilityResponse{
				BranchTimezone:  branch.Timezone,
				Date:            dateStr,
				DurationMinutes: svc.DurationMinutes,
				Slots:           []payload.SlotResponse{},
			}
			return nil
		}

		// 2. Load branch settings for slot step
		slotStep := 30
		bSettings, err := q.GetBranchSettings(ctx, branch.ID)
		if err == nil && bSettings.SlotStepMinutes > 0 {
			slotStep = int(bSettings.SlotStepMinutes)
		}

		// 3. Find eligible staff members
		var eligibleStaff []repository.ListStaffByBranchAndServiceRow
		if staffGUID != nil {
			sp, err := q.GetStaffProfileByGUID(ctx, pgtype.UUID{Bytes: *staffGUID, Valid: true})
			if err != nil || sp.Status != "ACTIVE" {
				return ErrStaffNotEligible
			}
			eligibleStaff = append(eligibleStaff, repository.ListStaffByBranchAndServiceRow{
				ID:          sp.ID,
				Guid:        sp.Guid,
				DisplayName: sp.DisplayName,
			})
		} else {
			eligibleStaff, err = q.ListStaffByBranchAndService(ctx, repository.ListStaffByBranchAndServiceParams{
				BranchID:  branch.ID,
				ServiceID: svc.ID,
			})
			if err != nil {
				return err
			}
		}

		type slotKey struct {
			start time.Time
			end   time.Time
		}
		slotStaffMap := make(map[slotKey][]uuid.UUID)

		dayStart := targetDate
		dayEnd := targetDate.Add(24 * time.Hour)

		// 4. For each staff member, evaluate availability
		for _, st := range eligibleStaff {
			sched, err := q.GetStaffScheduleForDay(ctx, repository.GetStaffScheduleForDayParams{
				StaffID:   st.ID,
				DayOfWeek: dayOfWeek,
			})
			if err != nil {
				continue
			}

			exceptions, err := q.GetScheduleExceptionsForStaffAndDate(ctx, repository.GetScheduleExceptionsForStaffAndDateParams{
				BranchID:  branch.ID,
				StaffID:   pgtype.Int8{Int64: st.ID, Valid: true},
				StartDate: pgtype.Date{Time: targetDate, Valid: true},
			})
			if err == nil && len(exceptions) > 0 {
				isOff := false
				for _, ex := range exceptions {
					if ex.ExceptionType == "HOLIDAY" || ex.ExceptionType == "LEAVE" {
						isOff = true
						break
					}
				}
				if isOff {
					continue
				}
			}

			existingApts, err := q.ListStaffAppointmentsForDateRange(ctx, repository.ListStaffAppointmentsForDateRangeParams{
				StaffID: st.ID,
				StartAt: pgtype.Timestamptz{Time: dayStart, Valid: true},
				EndAt:   pgtype.Timestamptz{Time: dayEnd, Valid: true},
			})
			if err != nil {
				return err
			}

			workStartMin := int(sched.StartTime.Microseconds / 60000000)
			workEndMin := int(sched.EndTime.Microseconds / 60000000)

			branchOpenMin := int(openMicro / 60000000)
			branchCloseMin := int(closeMicro / 60000000)

			actualStartMin := max(workStartMin, branchOpenMin)
			actualEndMin := min(workEndMin, branchCloseMin)

			serviceDur := int(svc.DurationMinutes)

			for startMin := actualStartMin; startMin+serviceDur <= actualEndMin; startMin += slotStep {
				endMin := startMin + serviceDur

				slotStartTime := targetDate.Add(time.Duration(startMin) * time.Minute)
				slotEndTime := targetDate.Add(time.Duration(endMin) * time.Minute)

				if sched.BreakStart.Valid && sched.BreakEnd.Valid {
					bStartMin := int(sched.BreakStart.Microseconds / 60000000)
					bEndMin := int(sched.BreakEnd.Microseconds / 60000000)
					if !(endMin <= bStartMin || startMin >= bEndMin) {
						continue
					}
				}

				hasOverlap := false
				for _, apt := range existingApts {
					if !(slotEndTime.Before(apt.StartAt.Time) || slotEndTime.Equal(apt.StartAt.Time) ||
						slotStartTime.After(apt.EndAt.Time) || slotStartTime.Equal(apt.EndAt.Time)) {
						hasOverlap = true
						break
					}
				}

				if !hasOverlap {
					key := slotKey{start: slotStartTime, end: slotEndTime}
					slotStaffMap[key] = append(slotStaffMap[key], st.Guid.Bytes)
				}
			}
		}

		var slots []payload.SlotResponse
		for k, staffList := range slotStaffMap {
			slots = append(slots, payload.SlotResponse{
				StartAt:    k.start,
				EndAt:      k.end,
				StaffGUIDs: staffList,
			})
		}

		sort.Slice(slots, func(i, j int) bool {
			return slots[i].StartAt.Before(slots[j].StartAt)
		})

		result = &payload.AvailabilityResponse{
			BranchTimezone:  branch.Timezone,
			Date:            dateStr,
			DurationMinutes: svc.DurationMinutes,
			Slots:           slots,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
