import { Link } from 'react-router-dom';
import downloadIcon from '../../assets/icon-download.svg';
import uploadIcon from '../../assets/icon-upload.svg';

export default function Beranda() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Title & Action Buttons */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
        <div>
          <h1 style={styles.headingTitle}>Selamat sore, Lidya</h1>
          <p style={styles.headingSubtitle}>
            Jumat, 16 Oktober 2026. Ibadah Minggu ini sudah terjadwal, ada 3 hal yang perlu dicek sebelum Sabtu siang.
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <button style={styles.btnSecondary}>
            <img src={downloadIcon} style={styles.iconSm} alt="Download" />
            Unduh template
          </button>
          <Link to="/impor-jadwal" style={styles.btnPrimary}>
            <img src={uploadIcon} style={{ ...styles.iconSm, filter: 'brightness(0) invert(1)' }} alt="Impor" />
            Impor jadwal
          </Link>
        </div>
      </div>

      {/* Stat Cards */}
      <div style={styles.statGrid}>
        <div style={styles.statCard}>
          <div style={styles.statLabel}>Jemaat terdaftar</div>
          <div style={styles.statNumber}>412</div>
          <div style={styles.statDesc}>8 jemaat baru bulan ini</div>
        </div>
        <div style={styles.statCard}>
          <div style={styles.statLabel}>Hadir Minggu lalu</div>
          <div style={styles.statNumber}>789</div>
          <div style={styles.statDesc}>Naik 4,0% dari 4 Okt (758)</div>
        </div>
        <div style={styles.statCard}>
          <div style={styles.statLabel}>Respons evaluasi</div>
          <div style={styles.statNumber}>64</div>
          <div style={styles.statDesc}>8,1% dari yang hadir, 11 Okt</div>
        </div>
        <div style={styles.statCard}>
          <div style={styles.statLabel}>Bentrok jadwal</div>
          <div style={{ ...styles.statNumber, color: '#B42238' }}>2</div>
          <div style={styles.statDesc}>Perlu diselesaikan sebelum Sab, 17 Okt</div>
        </div>
      </div>

      {/* Main Two Columns */}
      <div style={{ display: 'flex', gap: '20px', alignItems: 'flex-start' }}>
        {/* Kolom Kiri */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '20px' }}>
          {/* Card Banner Ibadah Minggu */}
          <div style={styles.bannerCard}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ color: '#C9952B', fontSize: '13px', fontWeight: '600' }}>Minggu, 18 Oktober</span>
              <span style={{ color: '#93AFB1', fontSize: '12px' }}>Google Form dibuka Min 06.00</span>
            </div>
            <div>
              <div style={{ fontSize: '28px', fontWeight: '800', lineHei: '34px' }}>Hidup yang Berbuah</div>
              <div style={{ color: '#93AFB1', fontSize: '13px', marginTop: '4px' }}>
                Yohanes 15:1–8, dibawakan Pdt. Samuel Effendi
              </div>
            </div>
            <div style={{ display: 'flex', gap: '12px' }}>
              <div style={styles.serviceItem}>
                <div style={styles.serviceItemHeader}>
                  <span style={{ fontSize: '15px', fontWeight: '600' }}>07.00</span>
                  <span style={styles.badgeSuccess}>Lengkap</span>
                </div>
                <div style={{ fontSize: '13px', fontWeight: '600' }}>Ibadah Umum 1</div>
                <div style={{ color: '#93AFB1', fontSize: '11px' }}>Petugas 14/14</div>
              </div>
              <div style={styles.serviceItem}>
                <div style={styles.serviceItemHeader}>
                  <span style={{ fontSize: '15px', fontWeight: '600' }}>09.30</span>
                  <span style={styles.badgeWarning}>2 kosong</span>
                </div>
                <div style={{ fontSize: '13px', fontWeight: '600' }}>Ibadah Umum 2</div>
                <div style={{ color: '#93AFB1', fontSize: '11px' }}>Petugas 12/14</div>
              </div>
              <div style={styles.serviceItem}>
                <div style={styles.serviceItemHeader}>
                  <span style={{ fontSize: '15px', fontWeight: '600' }}>09.30</span>
                  <span style={styles.badgeSuccess}>Lengkap</span>
                </div>
                <div style={{ fontSize: '13px', fontWeight: '600' }}>Sekolah Minggu</div>
                <div style={{ color: '#93AFB1', fontSize: '11px' }}>Petugas 6/6</div>
              </div>
            </div>
          </div>

          {/* Card Diagram Kehadiran */}
          <div style={styles.cardBase}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
              <div>
                <div style={{ fontSize: '15px', fontWeight: '600', color: '#0D1F21' }}>Kehadiran 4 minggu terakhir</div>
                <div style={{ color: '#66787A', fontSize: '13px', marginTop: '2px' }}>
                  Total Ibadah Umum 1, Umum 2, dan Sekolah Minggu
                </div>
              </div>
              <Link to="/absensi" style={{ color: '#00777C', fontSize: '13px', fontWeight: '600', textDecoration: 'none' }}>
                Lihat absensi
              </Link>
            </div>
            {/* Area Bar Chart */}
            <div style={{ display: 'flex', justifyContent: 'space-around', alignItems: 'flex-end', height: '180px', paddingBottom: '8px' }}>
              <div style={styles.barColumn}>
                <span style={styles.barValue}>781</span>
                <div style={{ ...styles.barTrack, height: '130px', backgroundColor: '#7CC3C5' }}></div>
                <span style={styles.barLabel}>20 Sep</span>
              </div>
              <div style={styles.barColumn}>
                <span style={styles.barValue}>742</span>
                <div style={{ ...styles.barTrack, height: '120px', backgroundColor: '#7CC3C5' }}></div>
                <span style={styles.barLabel}>27 Sep</span>
              </div>
              <div style={styles.barColumn}>
                <span style={styles.barValue}>810</span>
                <div style={{ ...styles.barTrack, height: '138px', backgroundColor: '#7CC3C5' }}></div>
                <span style={styles.barLabel}>4 Okt</span>
              </div>
              <div style={styles.barColumn}>
                <span style={{ ...styles.barValue, fontWeight: '600', color: '#0D1F21' }}>789</span>
                <div style={{ ...styles.barTrack, height: '132px', backgroundColor: '#009490' }}></div>
                <span style={styles.barLabel}>11 Okt</span>
              </div>
            </div>
          </div>
        </div>

        {/* Kolom Kanan */}
        <div style={{ width: '380px', display: 'flex', flexDirection: 'column', gap: '20px' }}>
          {/* Card Perlu Dicek */}
          <div style={styles.cardBase}>
            <div style={{ borderBottom: '1px solid #E1E8E8', paddingBottom: '12px' }}>
              <div style={{ fontSize: '15px', fontWeight: '600', color: '#0D1F21' }}>Perlu dicek</div>
              <div style={{ color: '#66787A', fontSize: '12px', marginTop: '2px' }}>3 item, batas Sab 17 Okt pukul 12.00</div>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column' }}></div>
          </div>

          {/* Card Aktivitas Terbaru */}
          <div style={styles.cardBase}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid #E1E8E8', paddingBottom: '12px' }}>
              <span style={{ fontSize: '15px', fontWeight: '600', color: '#0D1F21' }}>Aktivitas terbaru</span>
              <Link to="/audit-trail" style={{ color: '#00777C', fontSize: '12px', fontWeight: '600', textDecoration: 'none' }}>
                Audit trail
              </Link>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '14px', marginTop: '12px' }}></div>
          </div>
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
    textDecoration: 'none',
    padding: '9px 16px',
    backgroundColor: '#00777C',
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
  statGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(4, 1fr)',
    gap: '16px',
  },
  statCard: {
    backgroundColor: '#FFFFFF',
    padding: '20px',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
  },
  statLabel: {
    color: '#445759',
    fontSize: '13px',
  },
  statNumber: {
    fontSize: '28px',
    fontWeight: '700',
    color: '#0D1F21',
    margin: '4px 0 6px 0',
  },
  statDesc: {
    color: '#66787A',
    fontSize: '12px',
  },
  bannerCard: {
    backgroundColor: '#0D2B2E',
    borderRadius: '16px',
    padding: '24px',
    color: '#E8F1F1',
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  serviceItem: {
    flex: 1,
    backgroundColor: '#17403F',
    borderRadius: '10px',
    padding: '14px',
    display: 'flex',
    flexDirection: 'column',
    gap: '4px',
  },
  serviceItemHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  badgeSuccess: {
    padding: '2px 6px',
    backgroundColor: '#E0F3F2',
    color: '#075E62',
    borderRadius: '4px',
    fontSize: '11px',
    fontWeight: '600',
  },
  badgeWarning: {
    padding: '2px 6px',
    backgroundColor: '#FFF4D9',
    color: '#8F5F00',
    borderRadius: '4px',
    fontSize: '11px',
    fontWeight: '600',
  },
  cardBase: {
    backgroundColor: '#FFFFFF',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    padding: '20px',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
  },
  barColumn: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: '8px',
  },
  barValue: {
    fontSize: '12px',
    fontWeight: '500',
    color: '#66787A',
  },
  barTrack: {
    width: '56px',
    borderRadius: '6px 6px 0 0',
  },
  barLabel: {
    fontSize: '12px',
    color: '#66787A',
  },
  iconSm: {
    width: '18px',
    height: '18px',
  },
};