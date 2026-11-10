import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { CalendarCheck, Users, Activity, Sparkles } from "lucide-react";

export default function HomePage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-6 bg-brand-light text-brand-deep">
      {/* Status Badge */}
      <div className="mb-6">
        <Badge variant="mint" className="gap-2 px-3 py-1 text-xs">
          <Sparkles className="h-3.5 w-3.5 text-brand-primary" />
          Sistem Quevia Siap Digunakan
        </Badge>
      </div>

      {/* Hero Headline */}
      <h1 className="text-4xl font-extrabold tracking-tight sm:text-6xl max-w-3xl text-center text-brand-deep">
        Smart Queue &amp; Appointment Management
      </h1>

      <p className="mt-4 text-base sm:text-lg text-brand-deep/80 max-w-xl text-center font-medium">
        Platform SaaS Multi-Tenant dengan komponen shadcn/ui dan palet warna terstandarisasi untuk kenyamanan operasional.
      </p>

      {/* CTA Buttons with Brand Palette */}
      <div className="mt-8 flex flex-wrap items-center justify-center gap-4">
        <Button variant="brand" size="lg" className="gap-2">
          <CalendarCheck className="h-4 w-4" />
          Booking Janji Temu
        </Button>
        <Button
          variant="outline"
          size="lg"
          className="border-brand-primary text-brand-primary hover:bg-brand-mint/20"
        >
          <Users className="h-4 w-4 mr-2" />
          Ambil Antrean Walk-in
        </Button>
      </div>

      {/* Feature Cards showcasing Palette Tokens */}
      <div className="mt-12 grid grid-cols-1 sm:grid-cols-3 gap-6 max-w-4xl w-full">
        <Card className="border-brand-primary/20 bg-white/80 shadow-sm hover:shadow-md transition-shadow">
          <CardHeader className="pb-2">
            <Badge className="w-fit bg-brand-primary text-white">#2FA084</Badge>
            <CardTitle className="text-lg font-bold text-brand-deep mt-2">
              Brand Primary
            </CardTitle>
            <CardDescription>Aksi utama &amp; pemanggilan loket</CardDescription>
          </CardHeader>
          <CardContent className="text-sm text-zinc-600">
            Digunakan pada tombol utama, status pelayanan aktif, dan sorotan antrean prioritas.
          </CardContent>
        </Card>

        <Card className="border-brand-mint/40 bg-white/80 shadow-sm hover:shadow-md transition-shadow">
          <CardHeader className="pb-2">
            <Badge className="w-fit bg-brand-mint text-brand-deep font-semibold">
              #6FCF97
            </Badge>
            <CardTitle className="text-lg font-bold text-brand-deep mt-2">
              Mint Highlight
            </CardTitle>
            <CardDescription>Aksen &amp; status aktif</CardDescription>
          </CardHeader>
          <CardContent className="text-sm text-zinc-600">
            Memberikan visual lembut untuk tiket yang sedang dipanggil, notifikasi, dan badge sukses.
          </CardContent>
        </Card>

        <Card className="border-brand-deep/20 bg-white/80 shadow-sm hover:shadow-md transition-shadow">
          <CardHeader className="pb-2">
            <Badge className="w-fit bg-brand-deep text-white">#1F6F5F</Badge>
            <CardTitle className="text-lg font-bold text-brand-deep mt-2">
              Deep Pine Teal
            </CardTitle>
            <CardDescription>Tipografi &amp; struktur kontras</CardDescription>
          </CardHeader>
          <CardContent className="text-sm text-zinc-600">
            Warna dasar teks judul, sidebar navigasi, dan elemen struktural agar kontras memenuhi standar WCAG AA.
          </CardContent>
        </Card>
      </div>
    </main>
  );
}
