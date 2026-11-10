package main

import (
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func TestHealthz(t *testing.T) {
	h := server.Default()
	registerRoutes(h, nil, nil)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/healthz", nil)
	resp := w.Result()
	assert.DeepEqual(t, consts.StatusOK, resp.StatusCode())
}
