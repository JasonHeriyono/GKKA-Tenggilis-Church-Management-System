import userIcon from '../../assets/icon-user.svg';
import infoIcon from '../../assets/icon-info.svg';

export default function HakAkses() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Title & Action Button */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
        <div>
          <h1 style={styles.headingTitle}>Hak akses</h1>
          <p style={styles.headingSubtitle}>
            Empat peran bawaan. Setiap pengguna punya satu peran, dan Usher diberikan otomatis saat jemaat dijadwalkan bertugas.
          </p>
        </div>
        <button style={styles.btnPrimary}>
          <img
            src={userIcon}
            style={{ ...styles.iconSm, filter: 'brightness(0) invert(1)' }}
            alt="Atur"
          />
          Atur pengguna
        </button>
      </div>

      {/* Stat Cards Peran */}
      <div style={styles.roleGrid}>
        <div style={styles.roleCard}>
          <div style={styles.roleTitle}>Jemaat</div>
          <div style={styles.roleValue}>-</div>
          <div style={styles.roleDevice}>Aplikasi</div>
        </div>

        <div style={styles.roleCard}>
          <div style={styles.roleTitle}>Usher</div>
          <div style={styles.roleValue}>-</div>
          <div style={styles.roleDevice}>Aplikasi</div>
        </div>

        <div style={styles.roleCard}>
          <div style={styles.roleTitle}>Majelis</div>
          <div style={styles.roleValue}>-</div>
          <div style={styles.roleDevice}>Web, baca saja</div>
        </div>

        <div style={styles.roleCard}>
          <div style={styles.roleTitle}>Super Admin</div>
          <div style={styles.roleValue}>-</div>
          <div style={styles.roleDevice}>Web</div>
        </div>
      </div>

      {/* Card Matriks Hak Akses */}
      <div style={styles.matrixCard}>
        {/* Header Kolom Matriks */}
        <div style={styles.matrixHead}>
          <div style={{ width: '140px', ...styles.headCol }}>Modul</div>
          <div style={{ flex: 1, ...styles.headCol }}>Izin</div>
          <div style={{ width: '110px', textAlign: 'center', ...styles.roleHeadCol }}>Jemaat</div>
          <div style={{ width: '110px', textAlign: 'center', ...styles.roleHeadCol }}>Usher</div>
          <div style={{ width: '110px', textAlign: 'center', ...styles.roleHeadCol }}>Majelis</div>
          <div style={{ width: '110px', textAlign: 'center', ...styles.roleHeadCol }}>Super Admin</div>
        </div>

        {/* Area Baris Matriks Kosong */}
        <div style={styles.emptyContainer}>
          Belum ada konfigurasi hak akses modul yang dimuat
        </div>
      </div>

      {/* Notice Box Bawah */}
      <div style={styles.noticeBox}>
        <img
          src={infoIcon}
          style={{
            ...styles.iconSm,
            filter: 'invert(36%) sepia(91%) saturate(1450%) hue-rotate(25deg) brightness(92%) contrast(102%)',
          }}
          alt="Info"
        />
        <span style={styles.noticeText}>
          Peran tetap pada tahap MVP. Pengaturan izin per fitur (RBAC dinamis) termasuk pengembangan lanjutan.
        </span>
      </div>
    </div>
  );
}

const styles = {
  headingTitle: {
    fontSize: '28px',
    fontWeight: '800',
    color: '#0D1F21',
    lineHeight: '36px',
  },
  headingSubtitle: {
    color: '#445759',
    fontSize: '14px',
    marginTop: '4px',
    maxWidth: '680px',
  },
  btnPrimary: {
    padding: '9px 16px',
    backgroundColor: '#00777C',
    border: 'none',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '13px',
    fontWeight: '600',
    fontFamily: 'inherit',
    color: '#FFFFFF',
    cursor: 'pointer',
  },
  roleGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(4, 1fr)',
    gap: '16px',
  },
  roleCard: {
    padding: '18px',
    backgroundColor: '#FFFFFF',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  roleTitle: {
    color: '#0D1F21',
    fontSize: '16px',
    fontWeight: '600',
  },
  roleValue: {
    color: '#445759',
    fontSize: '13px',
  },
  roleDevice: {
    color: '#66787A',
    fontSize: '12px',
    fontWeight: '500',
  },
  matrixCard: {
    backgroundColor: '#FFFFFF',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    overflow: 'hidden',
    display: 'flex',
    flexDirection: 'column',
  },
  matrixHead: {
    padding: '12px 20px',
    backgroundColor: '#F7FAFA',
    display: 'flex',
    alignItems: 'center',
    gap: '16px',
    borderBottom: '1px solid #E1E8E8',
  },
  headCol: {
    color: '#66787A',
    fontSize: '12px',
    fontWeight: '500',
  },
  roleHeadCol: {
    color: '#0D1F21',
    fontSize: '13px',
    fontWeight: '600',
  },
  emptyContainer: {
    minHeight: '380px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#93AFB1',
    fontSize: '13px',
  },
  noticeBox: {
    padding: '14px 16px',
    backgroundColor: '#FBF2DD',
    borderRadius: '12px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  noticeText: {
    color: '#0D1F21',
    fontSize: '13px',
  },
  iconSm: {
    width: '18px',
    height: '18px',
  },
};