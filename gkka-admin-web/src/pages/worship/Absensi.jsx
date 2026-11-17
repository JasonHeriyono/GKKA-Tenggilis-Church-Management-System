import { useState } from 'react';

// Import aset SVG
import downloadIcon from '../../assets/icon-download.svg';
import plusIcon from '../../assets/icon-plus.svg';
import searchIcon from '../../assets/icon-search.svg';
import chevronDownIcon from '../../assets/icon-chevron-down.svg';
import historyIcon from '../../assets/icon-history.svg';
import checkIcon from '../../assets/icon-check.svg';

export default function Absensi() {
  const [activeEventTab, setActiveEventTab] = useState('Ibadah Umum 1, 11 Okt');
  const [activeFilter, setActiveFilter] = useState('Semua');
  const [keterangan, setKeterangan] = useState('lupa');

  const eventTabs = [
    'Ibadah Umum 1, 11 Okt',
    'Ibadah Umum 2, 11 Okt',
    'Persekutuan Doa, 14 Okt',
    'Komsel Rungkut, 16 Okt',
  ];

  const filterButtons = ['Semua', 'Usher', 'Manual', 'Belum hadir'];

  const handleSubmit = (e) => {
    e.preventDefault();
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Title & Action Buttons */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
        <div>
          <h1 style={styles.headingTitle}>Absensi Ibadah Umum 1</h1>
          <p style={styles.headingSubtitle}>Minggu, 11 Oktober 2026, 07.00 di Ruang Ibadah Utama</p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <button style={styles.btnSecondary}>
            <img src={downloadIcon} style={styles.iconSm} alt="Unduh" />
            Unduh Excel
          </button>
          <button style={styles.btnPrimary}>
            <img
              src={plusIcon}
              style={{ ...styles.iconSm, filter: 'brightness(0) invert(1)' }}
              alt="Input"
            />
            Input manual
          </button>
        </div>
      </div>

      {/* Tab Filter Event Ibadah */}
      <div style={styles.tabContainer}>
        {eventTabs.map((tab) => {
          const isActive = activeEventTab === tab;
          return (
            <div
              key={tab}
              onClick={() => setActiveEventTab(tab)}
              style={{
                ...styles.tabItem,
                backgroundColor: isActive ? '#FFFFFF' : 'transparent',
                color: isActive ? '#0D1F21' : '#445759',
                boxShadow: isActive ? '0 1px 2px rgba(0,0,0,0.05)' : 'none',
              }}
            >
              {tab}
            </div>
          );
        })}
      </div>

      {/* Stat Cards Ringkasan Kehadiran */}
      <div style={styles.statGrid}>
        <div style={styles.statCard}>
          <div style={styles.statNumber}>309</div>
          <div style={styles.statLabel}>Hadir tercatat</div>
        </div>
        <div style={styles.statCard}>
          <div style={{ display: 'flex', alignItems: 'flex-end', gap: '6px' }}>
            <div style={styles.statNumber}>291</div>
            <div style={{ ...styles.statLabel, marginBottom: '4px' }}>94%</div>
          </div>
          <div style={styles.statLabel}>Oleh usher di aplikasi</div>
        </div>
        <div style={styles.statCard}>
          <div style={{ display: 'flex', alignItems: 'flex-end', gap: '6px' }}>
            <div style={styles.statNumber}>18</div>
            <div style={{ ...styles.statLabel, marginBottom: '4px' }}>6%</div>
          </div>
          <div style={styles.statLabel}>Input manual sekretariat</div>
        </div>
        <div style={styles.statCard}>
          <div style={styles.statNumber}>4</div>
          <div style={styles.statLabel}>Usher bertugas</div>
        </div>
      </div>

      {/* Layout Utama: Tabel Log Kehadiran & Panel Form Input Manual */}
      <div style={{ display: 'flex', gap: '20px', alignItems: 'flex-start' }}>
        {/* Kolom Kiri: Tabel Kehadiran */}
        <div style={styles.tableCard}>
          {/* Filter Toolbar */}
          <div style={styles.filterToolbar}>
            <div style={styles.searchBox}>
              <img src={searchIcon} style={styles.iconSm} alt="Cari" />
              <input
                type="text"
                placeholder="Cari nama…"
                style={styles.searchInput}
              />
            </div>
            {filterButtons.map((filter) => {
              const isActive = activeFilter === filter;
              return (
                <button
                  key={filter}
                  onClick={() => setActiveFilter(filter)}
                  style={{
                    ...styles.filterBtn,
                    backgroundColor: isActive ? '#E0F3F2' : '#FFFFFF',
                    border: isActive ? '1px solid #00777C' : '1px solid #CBD6D6',
                    color: isActive ? '#075E62' : '#445759',
                  }}
                >
                  {filter}
                </button>
              );
            })}
          </div>

          {/* Header Kolom Tabel */}
          <div style={styles.tableHead}>
            <div style={{ flex: 1, ...styles.headCol }}>Nama</div>
            <div style={{ width: '130px', ...styles.headCol }}>Komunitas</div>
            <div style={{ width: '210px', ...styles.headCol }}>Dicatat oleh</div>
            <div style={{ width: '60px', textAlign: 'right', ...styles.headCol }}>Jam</div>
          </div>

          {/* Area Baris Tabel Kehadiran Kosong */}
          <div style={styles.emptyContainer}>
            Belum ada catatan kehadiran yang dimuat
          </div>

          {/* Paginasi Tabel */}
          <div style={styles.paginationRow}>
            <span style={{ color: '#66787A', fontSize: '12px', fontWeight: '500' }}>0 dari 0 kehadiran</span>
            <div style={{ display: 'flex', gap: '8px' }}>
              <button disabled style={styles.btnPagination}>Sebelumnya</button>
              <button disabled style={styles.btnPagination}>Berikutnya</button>
            </div>
          </div>
        </div>

        {/* Kolom Kanan: Card Form Input Manual */}
        <div style={styles.formCard}>
          <div style={styles.formCardHeader}>
            <div style={{ color: '#0D1F21', fontSize: '16px', fontWeight: '600' }}>Input manual</div>
            <div style={{ color: '#66787A', fontSize: '13px', marginTop: '2px' }}>Untuk kehadiran yang belum tercatat</div>
          </div>

          <form onSubmit={handleSubmit} style={styles.formBody}>
            {/* Input Jemaat */}
            <div style={styles.fieldGroup}>
              <label style={styles.fieldLabel}>Jemaat</label>
              <div style={styles.inputWrapper}>
                <img src={searchIcon} style={styles.iconSm} alt="Cari" />
                <input
                  type="text"
                  placeholder="Cari jemaat..."
                  style={styles.textInput}
                />
              </div>
            </div>

            {/* Input Kegiatan */}
            <div style={styles.fieldGroup}>
              <label style={styles.fieldLabel}>Kegiatan</label>
              <div style={styles.selectWrapper}>
                <span style={{ color: '#0D1F21', fontSize: '14px' }}>Ibadah Umum 1, Min 11 Okt</span>
                <img src={chevronDownIcon} style={styles.iconSm} alt="Pilih" />
              </div>
            </div>

            {/* Radio Option Keterangan */}
            <div style={styles.radioGroup}>
              <label style={styles.fieldLabel}>Keterangan</label>
              <label style={styles.radioLabel}>
                <input
                  type="radio"
                  name="keterangan"
                  value="lupa"
                  checked={keterangan === 'lupa'}
                  onChange={(e) => setKeterangan(e.target.value)}
                  style={{ accentColor: '#00777C' }}
                />
                Lupa dicatat usher
              </label>
              <label style={styles.radioLabel}>
                <input
                  type="radio"
                  name="keterangan"
                  value="terlambat"
                  checked={keterangan === 'terlambat'}
                  onChange={(e) => setKeterangan(e.target.value)}
                  style={{ accentColor: '#00777C' }}
                />
                Datang terlambat
              </label>
              <label style={styles.radioLabel}>
                <input
                  type="radio"
                  name="keterangan"
                  value="lainnya"
                  checked={keterangan === 'lainnya'}
                  onChange={(e) => setKeterangan(e.target.value)}
                  style={{ accentColor: '#00777C' }}
                />
                Lainnya
              </label>
            </div>

            {/* Note Audit Trail */}
            <div style={styles.auditNotice}>
              <img src={historyIcon} style={styles.iconSm} alt="Audit" />
              <span style={{ color: '#66787A', fontSize: '12px', fontWeight: '500' }}>
                Tercatat di audit trail atas nama pengguna aktif.
              </span>
            </div>

            {/* Tombol Submit */}
            <button type="submit" style={styles.btnSubmit}>
              <img
                src={checkIcon}
                style={{ width: '14px', height: '14px', filter: 'brightness(0) invert(1)' }}
                alt="Check"
              />
              Simpan kehadiran
            </button>
          </form>
        </div>
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
  tabContainer: {
    padding: '3px',
    backgroundColor: '#EEF3F3',
    borderRadius: '8px',
    display: 'inline-flex',
    gap: '2px',
    width: 'fit-content',
  },
  tabItem: {
    padding: '6px 12px',
    borderRadius: '6px',
    fontSize: '13px',
    fontWeight: '600',
    cursor: 'pointer',
    userSelect: 'none',
  },
  statGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(4, 1fr)',
    gap: '16px',
  },
  statCard: {
    padding: '18px',
    backgroundColor: '#FFFFFF',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
  },
  statNumber: {
    color: '#0D1F21',
    fontSize: '26px',
    fontWeight: '700',
  },
  statLabel: {
    color: '#66787A',
    fontSize: '12px',
    fontWeight: '500',
    marginTop: '2px',
  },
  tableCard: {
    flex: 1,
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
    width: '220px',
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
  filterBtn: {
    padding: '7px 12px',
    borderRadius: '8px',
    fontSize: '13px',
    fontWeight: '600',
    fontFamily: 'inherit',
    cursor: 'pointer',
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
    minHeight: '380px',
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
  formCard: {
    width: '340px',
    backgroundColor: '#FFFFFF',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    overflow: 'hidden',
    display: 'flex',
    flexDirection: 'column',
  },
  formCardHeader: {
    padding: '16px 20px',
    borderBottom: '1px solid #E1E8E8',
  },
  formBody: {
    padding: '20px',
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  fieldGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
  },
  fieldLabel: {
    color: '#445759',
    fontSize: '13px',
    fontWeight: '600',
  },
  inputWrapper: {
    padding: '10px 12px',
    backgroundColor: '#FFFFFF',
    borderRadius: '8px',
    border: '1px solid #CBD6D6',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  textInput: {
    border: 'none',
    outline: 'none',
    width: '100%',
    fontSize: '14px',
    fontFamily: 'inherit',
    color: '#0D1F21',
    backgroundColor: 'transparent',
  },
  selectWrapper: {
    padding: '10px 12px',
    backgroundColor: '#FFFFFF',
    borderRadius: '8px',
    border: '1px solid #CBD6D6',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    cursor: 'pointer',
  },
  radioGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: '10px',
  },
  radioLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    fontSize: '13px',
    color: '#0D1F21',
    cursor: 'pointer',
  },
  auditNotice: {
    padding: '10px 12px',
    backgroundColor: '#F7FAFA',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  btnSubmit: {
    padding: '10px 16px',
    backgroundColor: '#00777C',
    border: 'none',
    borderRadius: '8px',
    color: '#FFFFFF',
    fontSize: '14px',
    fontWeight: '600',
    fontFamily: 'inherit',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '8px',
  },
  iconSm: {
    width: '18px',
    height: '18px',
  },
};