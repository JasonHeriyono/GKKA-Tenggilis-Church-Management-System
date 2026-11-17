import { useState } from 'react';

import alertIcon from '../../assets/icon-alert.svg';
import calendarIcon from '../../assets/icon-calendar.svg';
import downloadIcon from '../../assets/icon-download.svg';
import starIcon from '../../assets/icon-star.svg';
import messageIcon from '../../assets/icon-message.svg';

export default function LaporanMajelis() {
  const [activeCommentTab, setActiveCommentTab] = useState('Semua');

  const commentTabs = ['Semua', 'Suara', 'Pujian', 'Waktu'];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Banner Peringatan Mode Baca Majelis */}
      <div style={styles.bannerWarning}>
        <img
          src={alertIcon}
          style={styles.iconAlert}
          alt="Peringatan"
        />
        <div style={styles.bannerText}>
          Mode baca. Menu Lain di gembok.
        </div>
      </div>

      {/* Title & Periode Selector */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
        <div>
          <h1 style={styles.headingTitle}>Evaluasi ibadah</h1>
          <p style={styles.headingSubtitle}>
            Ringkasan evaluasi ibadah jemaat per periode untuk monitoring Majelis.
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <button style={styles.btnSecondary}>
            <img src={calendarIcon} style={styles.iconSm} alt="Periode" />
            Pilih periode
          </button>
          <button style={styles.btnPrimary}>
            <img
              src={downloadIcon}
              style={{ ...styles.iconSm, filter: 'brightness(0) invert(1)' }}
              alt="Unduh"
            />
            Unduh Excel
          </button>
        </div>
      </div>

      {/* Stat Cards Ringkasan Nilai */}
      <div style={styles.statGrid}>
        <div style={styles.statCard}>
          <div style={styles.statHeaderRow}>
            <img src={starIcon} style={styles.iconSm} alt="Rating" />
            Rata-rata keseluruhan
          </div>
          <div style={styles.statNumber}>-</div>
          <div style={styles.statDesc}>Belum ada penilaian</div>
        </div>

        <div style={styles.statCard}>
          <div style={styles.statHeaderRow}>
            <img src={messageIcon} style={styles.iconSm} alt="Respons" />
            Respons
          </div>
          <div style={styles.statNumber}>0</div>
          <div style={styles.statDesc}>0% dari kehadiran</div>
        </div>

        <div style={styles.statCard}>
          <div style={styles.statHeaderRow}>
            <img src={alertIcon} style={styles.iconSm} alt="Aspek" />
            Aspek terendah
          </div>
          <div style={{ ...styles.statNumber, color: '#8F5F00', fontSize: '20px' }}>-</div>
          <div style={styles.statDesc}>Tidak ada catatan</div>
        </div>
      </div>

      {/* Layout Utama 2 Kolom */}
      <div style={{ display: 'flex', gap: '20px', alignItems: 'flex-start' }}>
        {/* Kolom Kiri */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '20px' }}>
          {/* Card Nilai per Aspek */}
          <div style={styles.cardBase}>
            <div style={styles.cardHeader}>
              <div style={styles.cardTitle}>Nilai per aspek</div>
              <div style={styles.cardSubtitle}>Skala 1–5 berdasarkan pertanyaan evaluasi</div>
            </div>
            <div style={styles.emptyContainerMd}>
              Belum ada data nilai per aspek yang dimuat
            </div>
          </div>

          {/* Card Per Minggu */}
          <div style={styles.cardBase}>
            <div style={styles.cardHeader}>
              <div style={styles.cardTitle}>Per minggu</div>
            </div>
            <div style={styles.tableHead}>
              <div style={{ width: '70px', ...styles.headCol }}>Minggu</div>
              <div style={{ flex: 1, ...styles.headCol }}>Tema</div>
              <div style={{ width: '160px', ...styles.headCol }}>Pembicara</div>
              <div style={{ width: '70px', textAlign: 'right', ...styles.headCol }}>Respons</div>
              <div style={{ width: '70px', textAlign: 'right', ...styles.headCol }}>Rata-rata</div>
              <div style={{ width: '60px', textAlign: 'right', ...styles.headCol }}>Suara</div>
            </div>
            <div style={styles.emptyContainerSm}>
              Belum ada rincian evaluasi mingguan
            </div>
          </div>
        </div>

        {/* Kolom Kanan: Komentar Jemaat */}
        <div style={styles.commentsCard}>
          <div style={styles.commentsHeader}>
            <div>
              <div style={styles.cardTitle}>Komentar jemaat</div>
              <div style={styles.cardSubtitle}>Saran tanpa nama pengisi</div>
            </div>
            <button style={styles.btnAllText}>Semua</button>
          </div>

          <div style={styles.commentTabWrapper}>
            <div style={styles.tabContainer}>
              {commentTabs.map((tab) => {
                const isActive = activeCommentTab === tab;
                return (
                  <div
                    key={tab}
                    onClick={() => setActiveCommentTab(tab)}
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
          </div>

          <div style={styles.emptyContainerLg}>
            Belum ada komentar atau saran jemaat
          </div>
        </div>
      </div>
    </div>
  );
}

const styles = {
  bannerWarning: {
    padding: '12px 16px',
    backgroundColor: '#FDE8EB',
    border: '1px solid #F8C8D0',
    borderRadius: '12px',
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  iconAlert: {
    width: '18px',
    height: '18px',
    filter: 'invert(19%) sepia(85%) saturate(3048%) hue-rotate(338deg) brightness(87%) contrast(90%)',
    flexShrink: 0,
  },
  bannerText: {
    flex: 1,
    color: '#B42238',
    fontSize: '13px',
    fontWeight: '500',
    lineHeight: '20px',
  },
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
  statGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(3, 1fr)',
    gap: '16px',
  },
  statCard: {
    padding: '20px',
    backgroundColor: '#FFFFFF',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  statHeaderRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    color: '#445759',
    fontSize: '13px',
  },
  statNumber: {
    color: '#0D1F21',
    fontSize: '26px',
    fontWeight: '700',
  },
  statDesc: {
    color: '#66787A',
    fontSize: '12px',
    fontWeight: '500',
  },
  cardBase: {
    backgroundColor: '#FFFFFF',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    overflow: 'hidden',
    display: 'flex',
    flexDirection: 'column',
  },
  cardHeader: {
    padding: '16px 20px',
    borderBottom: '1px solid #E1E8E8',
  },
  cardTitle: {
    color: '#0D1F21',
    fontSize: '16px',
    fontWeight: '600',
  },
  cardSubtitle: {
    color: '#66787A',
    fontSize: '13px',
    marginTop: '2px',
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
  emptyContainerSm: {
    minHeight: '200px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#93AFB1',
    fontSize: '13px',
  },
  emptyContainerMd: {
    minHeight: '220px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#93AFB1',
    fontSize: '13px',
  },
  emptyContainerLg: {
    minHeight: '380px',
    padding: '20px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#93AFB1',
    fontSize: '13px',
  },
  commentsCard: {
    width: '380px',
    backgroundColor: '#FFFFFF',
    borderRadius: '12px',
    border: '1px solid #E1E8E8',
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    overflow: 'hidden',
    display: 'flex',
    flexDirection: 'column',
  },
  commentsHeader: {
    padding: '16px 20px',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    borderBottom: '1px solid #E1E8E8',
  },
  btnAllText: {
    background: 'none',
    border: 'none',
    color: '#00777C',
    fontSize: '13px',
    fontWeight: '600',
    cursor: 'pointer',
  },
  commentTabWrapper: {
    padding: '12px 20px',
    borderBottom: '1px solid #E1E8E8',
  },
  tabContainer: {
    padding: '3px',
    backgroundColor: '#EEF3F3',
    borderRadius: '8px',
    display: 'inline-flex',
    gap: '4px',
  },
  tabItem: {
    padding: '6px 12px',
    borderRadius: '6px',
    fontSize: '12px',
    fontWeight: '600',
    cursor: 'pointer',
    userSelect: 'none',
  },
  iconSm: {
    width: '18px',
    height: '18px',
  },
};