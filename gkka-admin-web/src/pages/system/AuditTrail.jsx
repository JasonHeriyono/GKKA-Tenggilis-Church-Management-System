import downloadIcon from '../../assets/icon-download.svg';
import searchIcon from '../../assets/icon-search.svg';
import calendarIcon from '../../assets/icon-calendar.svg';
import lockIcon from '../../assets/icon-lock.svg';

export default function AuditTrail() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Title & Action Button */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
        <div>
          <h1 style={styles.headingTitle}>Audit trail</h1>
          <p style={styles.headingSubtitle}>
            Setiap perubahan data penting tercatat: siapa, apa, dan kapan. Catatan tidak bisa diubah atau dihapus.
          </p>
        </div>
        <button style={styles.btnSecondary}>
          <img src={downloadIcon} style={styles.iconSm} alt="Unduh" />
          Unduh Excel
        </button>
      </div>

      {/* Card Tabel Log Audit Trail */}
      <div style={styles.tableCard}>
        {/* Filter Toolbar */}
        <div style={styles.filterToolbar}>
          <div style={styles.searchBox}>
            <img src={searchIcon} style={styles.iconSm} alt="Cari" />
            <input
              type="text"
              placeholder="Cari nama, data, atau file…"
              style={styles.searchInput}
            />
          </div>
          <button style={styles.filterBtnWithIcon}>
            <img src={calendarIcon} style={styles.iconSm} alt="Rentang Waktu" />
            Rentang waktu
          </button>
          <button style={styles.filterBtnMuted}>
            Semua pengguna
          </button>
          <button style={styles.filterBtnMuted}>
            Semua jenis
          </button>
          <div style={{ flex: 1 }}></div>
          <span style={styles.countText}>0 catatan</span>
        </div>

        {/* Header Kolom Tabel */}
        <div style={styles.tableHead}>
          <div style={{ width: '80px', ...styles.headCol }}>Waktu</div>
          <div style={{ width: '170px', ...styles.headCol }}>Oleh</div>
          <div style={{ width: '100px', ...styles.headCol }}>Jenis</div>
          <div style={{ width: '250px', ...styles.headCol }}>Data</div>
          <div style={{ flex: 1, ...styles.headCol }}>Perubahan</div>
        </div>

        {/* Area Baris Log Kosong */}
        <div style={styles.emptyContainer}>
          Belum ada catatan aktivitas yang tercatat
        </div>

        {/* Paginasi Tabel */}
        <div style={styles.paginationRow}>
          <span style={styles.countText}>0 dari 0 catatan</span>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button disabled style={styles.btnPagination}>Sebelumnya</button>
            <button disabled style={styles.btnPagination}>Berikutnya</button>
          </div>
        </div>
      </div>

      {/* Keterangan Kebijakan Log */}
      <div style={styles.policyRow}>
        <img src={lockIcon} style={{ ...styles.iconSm, opacity: 0.6 }} alt="Keamanan" />
        <span style={styles.policyText}>
          Disimpan 5 tahun. Hanya Super Admin yang dapat membuka halaman ini.
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
  btnSecondary: {
    padding: '9px 16px',
    backgroundColor: '#FFFFFF',
    border: '1px solid #CBD6D6',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '13px',
    fontWeight: '600',
    fontFamily: 'inherit',
    color: '#0D1F21',
    cursor: 'pointer',
  },
  tableCard: {
    backgroundColor: '#FFFFFF',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    overflow: 'hidden',
    display: 'flex',
    flexDirection: 'column',
  },
  filterToolbar: {
    padding: '14px 20px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    borderBottom: '1px solid #E1E8E8',
  },
  searchBox: {
    width: '280px',
    padding: '8px 12px',
    backgroundColor: '#FFFFFF',
    borderRadius: '8px',
    border: '1px solid #CBD6D6',
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
  filterBtnWithIcon: {
    padding: '7px 12px',
    backgroundColor: '#FFFFFF',
    border: '1px solid #CBD6D6',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    color: '#0D1F21',
    fontSize: '13px',
    fontWeight: '600',
    fontFamily: 'inherit',
    cursor: 'pointer',
  },
  filterBtnMuted: {
    padding: '7px 12px',
    backgroundColor: '#FFFFFF',
    border: '1px solid #CBD6D6',
    borderRadius: '8px',
    color: '#445759',
    fontSize: '13px',
    fontWeight: '600',
    fontFamily: 'inherit',
    cursor: 'pointer',
  },
  countText: {
    color: '#66787A',
    fontSize: '12px',
    fontWeight: '500',
  },
  tableHead: {
    padding: '10px 20px',
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
  emptyContainer: {
    minHeight: '420px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#93AFB1',
    fontSize: '13px',
  },
  paginationRow: {
    padding: '12px 20px',
    borderTop: '1px solid #E1E8E8',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  btnPagination: {
    padding: '6px 12px',
    backgroundColor: '#FFFFFF',
    border: '1px solid #CBD6D6',
    borderRadius: '8px',
    fontSize: '13px',
    fontWeight: '600',
    color: '#93AFB1',
    cursor: 'not-allowed',
  },
  policyRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  policyText: {
    color: '#66787A',
    fontSize: '12px',
    fontWeight: '500',
  },
  iconSm: {
    width: '18px',
    height: '18px',
  },
};