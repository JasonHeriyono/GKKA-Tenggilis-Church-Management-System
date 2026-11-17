import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import MainLayout from './components/layout/MainLayout';
import ProtectedRoute from './components/layout/ProtectedRoute';

import Login from './pages/auth/Login';

import Beranda from './pages/dashboard/Beranda';

import JadwalKegiatan from './pages/worship/JadwalKegiatan';
import ImporJadwal from './pages/worship/ImporJadwal';
import PenugasanPelayan from './pages/worship/PenugasanPelayan';
import Absensi from './pages/worship/Absensi';

import DataJemaat from './pages/members/DataJemaat';
import KomunitasKeahlian from './pages/members/KomunitasKeahlian';

import Kuesioner from './pages/evaluation/kuesioner/Kuesioner';
import LaporanEvaluasi from './pages/evaluation/LaporanEvaluasi';
import LaporanMajelis from './pages/evaluation/LaporanMajelis';

import AuditTrail from './pages/system/AuditTrail';
import HakAkses from './pages/system/HakAkses';

const PagePlaceholder = ({ title }) => (
  <div style={{ padding: '24px', background: '#fff', borderRadius: '12px', border: '1px solid #E1E8E8' }}>
    <h2 style={{ fontSize: '20px', fontWeight: '700', color: '#0D1F21' }}>{title}</h2>
    <p style={{ color: '#66787A', fontSize: '13px', marginTop: '6px' }}>Halaman sedang dalam tahap konversi.</p>
  </div>
);

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />

        {/* Rute Umum yang Wajib Login */}
        <Route element={<ProtectedRoute />}>
          <Route element={<MainLayout />}>
            <Route path="/" element={<Beranda />} />
            <Route path="/jadwal-kegiatan" element={<JadwalKegiatan />} />
            <Route path="/impor-jadwal" element={<ImporJadwal />} />
            <Route path="/penugasan-pelayan" element={<PenugasanPelayan />} />
            <Route path="/absensi" element={<Absensi />} />
            <Route path="/data-jemaat" element={<DataJemaat />} />
            <Route path="/komunitas-keahlian" element={<KomunitasKeahlian />} />
            <Route path="/kuesioner" element={<Kuesioner />} />
            <Route path="/audit-trail" element={<AuditTrail />} />
            <Route path="/hak-akses" element={<HakAkses />} />

            {/* Khusus Super Admin / Admin */}
            <Route element={<ProtectedRoute allowedRoles={['Super Admin']} />}>
              <Route path="/laporan-evaluasi" element={<LaporanEvaluasi />} />
            </Route>

            {/* Khusus Majelis */}
            <Route element={<ProtectedRoute allowedRoles={['Majelis']} />}>
              <Route path="/laporan-majelis" element={<LaporanMajelis />} />
            </Route>
          </Route>
        </Route>

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  );
}