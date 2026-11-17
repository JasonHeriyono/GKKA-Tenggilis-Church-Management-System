import { useLocation } from 'react-router-dom';
import searchIcon from '../../assets/icon-search.svg';
import bellIcon from '../../assets/icon-bell.svg';

// Pemetaan rute ke judul & kategori breadcrumb
const routeMeta = {
  '/': { title: 'Beranda' },
  '/jadwal-kegiatan': { category: 'Ibadah & pelayanan', title: 'Jadwal kegiatan' },
  '/impor-jadwal': { category: 'Ibadah & pelayanan', title: 'Impor Excel' },
  '/penugasan-pelayan': { category: 'Ibadah & pelayanan', title: 'Penugasan pelayan' },
  '/absensi': { category: 'Ibadah & pelayanan', title: 'Absensi' },
  '/data-jemaat': { category: 'Jemaat', title: 'Data jemaat' },
  '/komunitas-keahlian': { category: 'Jemaat', title: 'Komunitas & keahlian' },
  '/kuesioner': { category: 'Evaluasi', title: 'Kuesioner' },
  '/laporan-evaluasi': { category: 'Evaluasi', title: 'Laporan evaluasi' },
  '/laporan-majelis': { category: 'Evaluasi', title: 'Laporan evaluasi' },
  '/audit-trail': { category: 'Sistem', title: 'Audit trail' },
  '/hak-akses': { category: 'Sistem', title: 'Hak akses' },
};

export default function Topbar() {
  const location = useLocation();
  const current = routeMeta[location.pathname] || { title: 'GKKA Tenggilis' };

  return (
    <header style={styles.topbar}>
      {/* Area Judul / Breadcrumb Dinamis */}
      <div>
        {current.category ? (
          <div style={styles.breadcrumb}>
            <span style={styles.categoryText}>{current.category}</span>
            <span style={styles.separator}>/</span>
            <span style={styles.activeTitle}>{current.title}</span>
          </div>
        ) : (
          <span style={styles.singleTitle}>{current.title}</span>
        )}
      </div>

      {/* Bagian Kanan (Search Bar & Notifikasi) */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <div style={styles.searchBox}>
          <img src={searchIcon} style={styles.iconSm} alt="Cari" />
          <input
            type="text"
            placeholder="Cari jemaat, kegiatan, atau data…"
            style={styles.searchInput}
          />
        </div>
        <div style={styles.bellButton}>
          <img src={bellIcon} style={styles.iconSm} alt="Notifikasi" />
        </div>
      </div>
    </header>
  );
}

const styles = {
  // Ubah objek styles.topbar di Topbar.jsx menjadi:
  topbar: {
    padding: '14px 32px',
    backgroundColor: '#FFFFFF',
    borderBottom: '1px solid #E1E8E8',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: '16px',
    position: 'sticky',
    top: 0,
    zIndex: 10,
    minWidth: '960px', /* Selaras dengan batas konten bawah */
    boxSizing: 'border-box',
  },
  singleTitle: {
    color: '#0D1F21',
    fontSize: '14px',
    fontWeight: '600',
  },
  breadcrumb: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '13px',
  },
  categoryText: {
    color: '#66787A',
  },
  separator: {
    color: '#66787A',
  },
  activeTitle: {
    color: '#0D1F21',
    fontWeight: '600',
  },
  searchBox: {
    width: '340px',
    padding: '8px 12px',
    backgroundColor: '#FFFFFF',
    border: '1px solid #CBD6D6',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  searchInput: {
    border: 'none',
    outline: 'none',
    width: '100%',
    fontSize: '13px',
    fontFamily: 'inherit',
    color: '#0D1F21',
    backgroundColor: 'transparent',
  },
  bellButton: {
    width: '36px',
    height: '36px',
    border: '1px solid #E1E8E8',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    cursor: 'pointer',
    backgroundColor: '#FFFFFF',
  },
  iconSm: {
    width: '18px',
    height: '18px',
  },
};