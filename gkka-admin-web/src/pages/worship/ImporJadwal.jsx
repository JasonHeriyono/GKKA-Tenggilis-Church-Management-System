import { useState, useRef } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import apiClient from '../../services/api';

import downloadIcon from '../../assets/icon-download.svg';
import sheetIcon from '../../assets/icon-sheet.svg';
import uploadIcon from '../../assets/icon-upload.svg';
import infoIcon from '../../assets/icon-info.svg';
import checkIcon from '../../assets/icon-check.svg';

export default function ImporJadwal() {
  const [selectedFile, setSelectedFile] = useState(null);
  const [activeTab, setActiveTab] = useState('Semua');
  const fileInputRef = useRef(null);
  const navigate = useNavigate();

  const [previewData, setPreviewData] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  const tabs = ['Semua', 'Perlu diperbaiki', 'Baru', 'Diperbarui', 'Tidak berubah'];

  const handleFileChange = async (e) => {
    const file = e.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      await fetchPreview(file);
    }
  };

  const handlePickFileClick = () => {
    fileInputRef.current?.click();
  };

  // --- Fungsi Load Preview Excel ---
  const fetchPreview = async (file) => {
    setIsLoading(true);
    setErrorMsg('');
    setPreviewData(null);
    setActiveTab('Semua');

    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await apiClient.post('/schedules/import/preview', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      setPreviewData(response.data.data);
    } catch (err) {
      console.error('Gagal pratinjau:', err);
      setErrorMsg(err.response?.data?.message || 'Gagal memeriksa file jadwal.');
    } finally {
      setIsLoading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  // --- Fungsi Simpan Excel ke Database ---
  const handleSaveImport = async () => {
    if (!selectedFile || !previewData?.can_commit) return;
    
    setIsLoading(true);
    const formData = new FormData();
    formData.append('file', selectedFile);

    try {
      const response = await apiClient.post('/schedules/import', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      alert(`Jadwal berhasil disimpan!\nDitambah: ${response.data.data.ditambah}\nDiperbarui: ${response.data.data.diperbarui}`);
      navigate('/jadwal-kegiatan'); // Arahkan kembali ke tabel jadwal
    } catch (err) {
      console.error('Gagal menyimpan:', err);
      alert(err.response?.data?.message || 'Gagal menyimpan jadwal ke database.');
    } finally {
      setIsLoading(false);
    }
  };

  const getFilteredRows = () => {
    if (!previewData?.rows) return [];
    if (activeTab === 'Perlu diperbaiki') {
      return previewData.rows.filter(row => row.errors && Object.keys(row.errors).length > 0);
    }
    return previewData.rows;
  };

  const filteredRows = getFilteredRows();

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
        <div>
          <h1 style={styles.headingTitle}>Impor jadwal kegiatan</h1>
          <p style={styles.headingSubtitle}>
            Unggah file jadwal Excel untuk menambahkan kegiatan baru dan memperbarui penugasan.
          </p>
        </div>
        <button style={styles.btnSecondary}>
          <img src={downloadIcon} style={styles.iconSm} alt="Download" />
          Unduh template
        </button>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <div style={previewData ? styles.stepperCircleInactive : styles.stepperCircleActive}>1</div>
          <span style={{ fontSize: '13px', fontWeight: '600', color: previewData ? '#66787A' : '#0D1F21' }}>Unggah file</span>
        </div>
        <div style={styles.stepperLine}></div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <div style={previewData ? styles.stepperCircleActive : styles.stepperCircleInactive}>2</div>
          <span style={{ fontSize: '13px', fontWeight: '600', color: previewData ? '#0D1F21' : '#66787A' }}>Periksa hasil</span>
        </div>
        <div style={styles.stepperLine}></div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <div style={styles.stepperCircleInactive}>3</div>
          <span style={{ fontSize: '13px', fontWeight: '600', color: '#66787A' }}>Simpan</span>
        </div>
      </div>

      <div style={{ display: 'flex', gap: '16px', alignItems: 'stretch' }}>
        <div style={styles.fileCard}>
          <div style={styles.fileIconBox}>
            <img src={sheetIcon} style={{ width: '22px', height: '22px' }} alt="Excel File" />
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={styles.fileTitle}>
              {selectedFile ? selectedFile.name : 'Belum ada file dipilih'}
            </div>
            <div style={styles.fileSubtitle}>Format .xlsx atau .xls</div>
          </div>
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileChange}
            accept=".xlsx, .xls"
            style={{ display: 'none' }}
          />
          <button onClick={handlePickFileClick} disabled={isLoading} style={styles.btnPickFile}>
            <img src={uploadIcon} style={styles.iconSm} alt="Pilih File" />
            {isLoading ? 'Memeriksa...' : 'Pilih file'}
          </button>
        </div>

        <div style={styles.statsSummaryCard}>
          <div style={styles.summaryCol}>
            <div style={{ color: '#075E62', ...styles.summaryValue }}>
              {previewData ? previewData.valid_rows : '-'}
            </div>
            <div style={styles.summaryLabel}>Valid (Baru/Diperbarui)</div>
          </div>
          <div style={{ ...styles.summaryCol, borderRight: 'none' }}>
            <div style={{ color: '#B42238', ...styles.summaryValue }}>
              {previewData ? previewData.invalid_rows : '-'}
            </div>
            <div style={styles.summaryLabel}>Perlu diperbaiki</div>
          </div>
        </div>
      </div>

      {errorMsg && (
        <div style={{ padding: '12px', backgroundColor: '#FFE5E5', color: '#D8000C', borderRadius: '8px', fontSize: '13px' }}>
          <strong>Error: </strong> {errorMsg}
        </div>
      )}

      <div style={styles.tableCard}>
        <div style={{ padding: '12px 20px' }}>
          <div style={styles.tabContainer}>
            {tabs.map((tab) => {
              const isActive = activeTab === tab;
              let tabCount = 0;
              if (previewData) {
                if (tab === 'Semua') tabCount = previewData.total_rows;
                if (tab === 'Perlu diperbaiki') tabCount = previewData.invalid_rows;
              }

              return (
                <div
                  key={tab}
                  onClick={() => {
                    if (previewData && (tab === 'Semua' || tab === 'Perlu diperbaiki')) {
                      setActiveTab(tab);
                    }
                  }}
                  style={{
                    ...styles.tabItem,
                    backgroundColor: isActive ? '#FFFFFF' : 'transparent',
                    color: isActive ? '#0D1F21' : '#445759',
                    boxShadow: isActive ? '0 1px 2px rgba(0,0,0,0.05)' : 'none',
                    cursor: previewData ? 'pointer' : 'not-allowed',
                    opacity: previewData ? 1 : 0.6
                  }}
                >
                  {tab} <span style={{ color: '#66787A' }}>{tabCount}</span>
                </div>
              );
            })}
          </div>
        </div>

        <div style={{ height: '1px', backgroundColor: '#E1E8E8' }}></div>

        {/* Kolom Table Disesuaikan dengan "template_kegiatan.xlsx" */}
        <div style={styles.tableHead}>
          <div style={{ width: '44px', ...styles.headCol }}>Baris</div>
          <div style={{ width: '96px', ...styles.headCol }}>Tanggal</div>
          <div style={{ width: '100px', ...styles.headCol }}>Waktu</div>
          <div style={{ width: '180px', ...styles.headCol }}>Nama Kegiatan</div>
          <div style={{ flex: 1, ...styles.headCol }}>Tema & Pembicara</div>
          <div style={{ width: '110px', ...styles.headCol }}>Komunitas</div>
          <div style={{ width: '200px', ...styles.headCol }}>Status / Error</div>
        </div>

        {!previewData && !isLoading && (
          <div style={styles.emptyContainer}>Silakan unggah file Excel untuk melihat pratinjau jadwal</div>
        )}
        {isLoading && (
          <div style={styles.emptyContainer}>Sedang membaca dan memvalidasi file...</div>
        )}
        {previewData && filteredRows.length === 0 && (
          <div style={styles.emptyContainer}>Tidak ada data untuk kategori ini.</div>
        )}

        {previewData && (
          <div style={{ display: 'flex', flexDirection: 'column', maxHeight: '400px', overflowY: 'auto' }}>
            {filteredRows.map((row, idx) => {
              const hasError = row.errors && Object.keys(row.errors).length > 0;
              return (
                <div key={idx} style={{ 
                  ...styles.tableRow, 
                  backgroundColor: hasError ? '#FFF4F4' : '#FFFFFF',
                  borderLeft: hasError ? '3px solid #D8000C' : '3px solid transparent'
                }}>
                  <div style={{ width: '44px', color: '#66787A', fontSize: '13px' }}>{row.line}</div>
                  <div style={{ width: '96px', color: '#0D1F21', fontSize: '13px' }}>{row.tanggal || '-'}</div>
                  
                  {/* Waktu (Mulai - Selesai) */}
                  <div style={{ width: '100px', color: '#0D1F21', fontSize: '13px' }}>
                    {row.waktu_mulai ? row.waktu_mulai.substring(0, 5) : '-'} 
                    {row.waktu_selesai ? ` - ${row.waktu_selesai.substring(0, 5)}` : ''}
                  </div>

                  <div style={{ width: '180px', color: '#0D1F21', fontSize: '13px', fontWeight: '500' }}>
                    {row.nama_kegiatan || '-'}
                  </div>
                  
                  {/* Tema & Pembicara */}
                  <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '2px' }}>
                    <span style={{ color: '#0D1F21', fontSize: '13px' }}>{row.tema || '-'}</span>
                    {row.pembicara && <span style={{ color: '#66787A', fontSize: '11px' }}>Oleh: {row.pembicara}</span>}
                  </div>
                  
                  {/* Komunitas */}
                  <div style={{ width: '110px', color: '#0D1F21', fontSize: '13px' }}>
                    {row.komunitas || 'Umum'}
                  </div>

                  {/* Errors */}
                  <div style={{ width: '200px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
                    {hasError ? (
                      Object.entries(row.errors).map(([col, msg]) => (
                        <div key={col} style={{ color: '#D8000C', fontSize: '12px' }}>
                          <strong>{col}:</strong> {msg}
                        </div>
                      ))
                    ) : (
                      <span style={{ color: '#075E62', fontSize: '12px', fontWeight: '500' }}>✓ Valid</span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        <div style={styles.paginationRow}>
          <div style={{ color: '#66787A', fontSize: '12px', fontWeight: '500' }}>
            {previewData ? filteredRows.length : 0} dari {previewData ? previewData.total_rows : 0} baris
          </div>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button disabled style={styles.btnPagination}>Sebelumnya</button>
            <button disabled style={styles.btnPagination}>Berikutnya</button>
          </div>
        </div>
      </div>

      <div style={styles.bottomActionBar}>
        <img src={infoIcon} style={{ width: '20px', height: '20px', filter: 'brightness(0) invert(1)', opacity: 0.8 }} alt="Info" />
        <div style={{ flex: 1 }}>
          <div style={{ color: '#E8F1F1', fontSize: '13px', fontWeight: '600' }}>
            {previewData ? (previewData.can_commit ? 'File siap disimpan' : 'Perbaiki error di Excel sebelum menyimpan') : 'Menunggu file Excel'}
          </div>
          <div style={{ color: '#93AFB1', fontSize: '12px' }}>
            {previewData ? 'Pastikan data kegiatan benar sebelum menyetujui.' : 'Pastikan kolom sesuai format template sebelum menyimpan.'}
          </div>
        </div>
        <Link to="/jadwal-kegiatan" style={styles.btnCancel}>Batal</Link>
        <button 
          disabled={!previewData || !previewData.can_commit || isLoading} 
          onClick={handleSaveImport}
          style={previewData?.can_commit ? styles.btnSaveActive : styles.btnSaveDisabled}
        >
          <img src={checkIcon} style={{ width: '14px', height: '14px', opacity: previewData?.can_commit ? 1 : 0.5, filter: previewData?.can_commit ? 'brightness(0) invert(1)' : 'none' }} alt="Simpan" />
          {isLoading ? 'Menyimpan...' : 'Simpan jadwal'}
        </button>
      </div>
    </div>
  );
}

const styles = {
  headingTitle: { fontSize: '28px', fontWeight: '800', color: '#0D1F21', lineHeight: '36px', margin: 0 },
  headingSubtitle: { color: '#445759', fontSize: '14px', marginTop: '4px', maxWidth: '680px' },
  btnSecondary: { padding: '9px 16px', backgroundColor: '#FFFFFF', border: '1px solid #CBD6D6', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', fontWeight: '600', fontFamily: 'inherit', color: '#0D1F21', cursor: 'pointer' },
  stepperCircleActive: { width: '28px', height: '28px', backgroundColor: '#00777C', color: 'white', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '13px', fontWeight: '600' },
  stepperCircleInactive: { width: '28px', height: '28px', backgroundColor: '#EEF3F3', color: '#66787A', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '13px', fontWeight: '600' },
  stepperLine: { width: '48px', height: '2px', backgroundColor: '#CBD6D6', borderRadius: '1px' },
  fileCard: { width: '440px', padding: '16px', backgroundColor: '#FFFFFF', borderRadius: '12px', border: '1px solid #E1E8E8', boxShadow: '0 1px 3px rgba(0,0,0,0.04)', display: 'flex', alignItems: 'center', gap: '14px', boxSizing: 'border-box' },
  fileIconBox: { width: '44px', height: '44px', backgroundColor: '#EEF3F3', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center' },
  fileTitle: { fontSize: '14px', fontWeight: '600', color: '#0D1F21', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' },
  fileSubtitle: { fontSize: '12px', color: '#66787A', marginTop: '2px' },
  btnPickFile: { padding: '6px 12px', backgroundColor: '#FFFFFF', border: '1px solid #CBD6D6', borderRadius: '8px', fontSize: '13px', fontWeight: '600', fontFamily: 'inherit', color: '#0D1F21', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '6px' },
  statsSummaryCard: { flex: 1, backgroundColor: '#FFFFFF', borderRadius: '12px', border: '1px solid #E1E8E8', boxShadow: '0 1px 3px rgba(0,0,0,0.04)', display: 'flex' },
  summaryCol: { flex: 1, padding: '14px 20px', borderRight: '1px solid #E1E8E8' },
  summaryValue: { fontSize: '26px', fontWeight: '700' },
  summaryLabel: { color: '#66787A', fontSize: '12px', fontWeight: '500' },
  tableCard: { backgroundColor: '#FFFFFF', borderRadius: '12px', border: '1px solid #E1E8E8', boxShadow: '0 1px 3px rgba(0,0,0,0.04)', overflow: 'hidden', display: 'flex', flexDirection: 'column' },
  tabContainer: { padding: '3px', backgroundColor: '#EEF3F3', borderRadius: '8px', display: 'inline-flex', gap: '2px' },
  tabItem: { padding: '6px 12px', borderRadius: '6px', fontSize: '13px', fontWeight: '600', display: 'flex', gap: '6px', userSelect: 'none' },
  tableHead: { padding: '10px 20px', backgroundColor: '#F7FAFA', display: 'flex', alignItems: 'center', gap: '16px', borderBottom: '1px solid #E1E8E8' },
  headCol: { color: '#66787A', fontSize: '12px', fontWeight: '500' },
  tableRow: { padding: '12px 20px', borderBottom: '1px solid #E1E8E8', display: 'flex', alignItems: 'flex-start', gap: '16px' },
  emptyContainer: { minHeight: '240px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#93AFB1', fontSize: '13px' },
  paginationRow: { padding: '12px 20px', borderTop: '1px solid #E1E8E8', display: 'flex', alignItems: 'center', justifyContent: 'space-between' },
  btnPagination: { padding: '6px 12px', backgroundColor: '#FFFFFF', border: '1px solid #CBD6D6', borderRadius: '8px', fontSize: '13px', fontWeight: '600', color: '#93AFB1', cursor: 'not-allowed' },
  bottomActionBar: { padding: '14px 20px', backgroundColor: '#0D2B2E', boxShadow: '0px 12px 32px -8px rgba(12, 30, 33, 0.12)', borderRadius: '12px', display: 'flex', alignItems: 'center', gap: '12px' },
  btnCancel: { textDecoration: 'none', padding: '10px 16px', backgroundColor: '#FFFFFF', borderRadius: '8px', color: '#0D1F21', fontSize: '14px', fontWeight: '600', cursor: 'pointer' },
  btnSaveDisabled: { padding: '10px 16px', backgroundColor: '#17403F', border: 'none', borderRadius: '8px', color: '#93AFB1', fontSize: '14px', fontWeight: '600', display: 'flex', alignItems: 'center', gap: '8px', cursor: 'not-allowed' },
  btnSaveActive: { padding: '10px 16px', backgroundColor: '#00777C', border: 'none', borderRadius: '8px', color: '#FFFFFF', fontSize: '14px', fontWeight: '600', display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' },
  iconSm: { width: '18px', height: '18px' },
};