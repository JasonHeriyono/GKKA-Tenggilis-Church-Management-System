import { useEffect, useRef } from 'react';
import { Link } from 'react-router-dom';
import useJadwalKegiatan from './useJadwalKegiatan';

import plusIcon from '../../assets/icon-plus.svg';
import uploadIcon from '../../assets/icon-upload.svg';
import leftIcon from '../../assets/icon-chevron-left.svg';
import rightIcon from '../../assets/icon-chevron-right.svg';
import refreshIcon from '../../assets/icon-refresh.svg';
import editIcon from '../../assets/icon-edit.svg';
import peopleIcon from '../../assets/icon-user-check.svg';
import closeIcon from '../../assets/icon-x.svg';
import calendarIcon from '../../assets/icon-calendar.svg';

function Icon({ src }) {
  return <img src={src} alt="" style={{ width: 17, height: 17 }} />;
}

function readableDate(value) {
  return new Date(`${value}T12:00:00`).toLocaleDateString('id-ID', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });
}

function ActivityModal({ state }) {
  const {
    form, changeField, modal, closeModal, communities,
    expanded, setExpanded, formError, fieldErrors, busy, save,
  } = state;

  const modalRef = useRef(null);
  const busyRef = useRef(busy);
  busyRef.current = busy;

  useEffect(() => {
    const previousFocus = document.activeElement;
    modalRef.current?.querySelector('input')?.focus();

    return () => {
      if (previousFocus?.isConnected) previousFocus.focus();
    };
  }, []);

  function handleKeys(event) {
    if (event.key === 'Escape') {
      event.preventDefault();
      if (!busyRef.current) closeModal();
    }

    if (event.key !== 'Tab') return;

    const elements = Array.from(
      modalRef.current.querySelectorAll(
        'button, input, select, textarea, [tabindex="0"]',
      ),
    ).filter(element => !element.matches(':disabled') && element.offsetParent);

    if (elements.length === 0) {
      event.preventDefault();
      modalRef.current.focus();
      return;
    }

    const first = elements[0];
    const last = elements[elements.length - 1];

    if (event.shiftKey && (document.activeElement === first ||
      document.activeElement === modalRef.current)) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  function field(key, label, type = 'text', maxLength) {
    return (
      <label key={key} style={styles.label}>
        {label}
        <input
          id={`kegiatan-${key}`}
          type={type}
          value={form[key]}
          required={['nama_kegiatan', 'tanggal', 'waktu_mulai'].includes(key)}
          maxLength={maxLength}
          aria-invalid={Boolean(fieldErrors[key])}
          aria-describedby={fieldErrors[key] ? `error-${key}` : undefined}
          onChange={event => changeField(key, event.target.value)}
          style={styles.input}
        />
        {fieldErrors[key] && (
          <span id={`error-${key}`} style={styles.fieldError}>
            {fieldErrors[key]}
          </span>
        )}
      </label>
    );
  }

  return (
    <div style={styles.overlay}>
      <section
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="activity-modal-title"
        tabIndex={-1}
        onKeyDown={handleKeys}
        style={styles.modal}
      >
        <div style={styles.header}>
          <h2 id="activity-modal-title" style={styles.sectionTitle}>
            {modal === 'edit' ? 'Edit kegiatan' : 'Tambah kegiatan'}
          </h2>
          <button
            type="button"
            aria-label="Tutup form"
            disabled={busy}
            onClick={closeModal}
            style={styles.secondary}
          >
            <Icon src={closeIcon} />
          </button>
        </div>

        {formError && <div role="alert" style={styles.error}>{formError}</div>}

        {Object.keys(fieldErrors).length > 0 && (
          <div role="alert" style={styles.error}>
            {Object.entries(fieldErrors).map(([key, value]) => (
              <div key={key}>{key}: {value}</div>
            ))}
          </div>
        )}

        <form onSubmit={save}>
          <fieldset disabled={busy} style={styles.fieldset}>
            {field('nama_kegiatan', 'Nama kegiatan *', 'text', 150)}

            <div style={styles.grid}>
              {field('tanggal', 'Tanggal *', 'date')}
              {field('waktu_mulai', 'Jam mulai *', 'time')}
            </div>

            <label style={styles.checkbox}>
              <input
                type="checkbox"
                checked={form.perlu_evaluasi}
                onChange={event =>
                  changeField('perlu_evaluasi', event.target.checked)
                }
              />
              Kegiatan ini memerlukan evaluasi
            </label>

            <button
              type="button"
              aria-expanded={expanded}
              aria-controls="activity-extra-fields"
              style={{ ...styles.secondary, alignSelf: 'flex-start' }}
              onClick={() => setExpanded(value => !value)}
            >
              {expanded ? 'Sembunyikan informasi tambahan' : 'Informasi tambahan'}
            </button>

            {expanded && (
              <div id="activity-extra-fields" style={styles.grid}>
                {field('waktu_selesai', 'Jam selesai', 'time')}

                <label style={styles.label}>
                  Komunitas
                  <select
                    value={form.id_komunitas}
                    onChange={event =>
                      changeField('id_komunitas', event.target.value)
                    }
                    style={styles.input}
                  >
                    <option value="">Umum</option>

                    {form.id_komunitas &&
                      !communities.some(item =>
                        String(item.id) === form.id_komunitas,
                      ) && (
                        <option value={form.id_komunitas}>
                          Komunitas saat ini — ID {form.id_komunitas}
                        </option>
                      )}

                    {communities.map(item => (
                      <option key={item.id} value={String(item.id)}>
                        {item.nama}
                      </option>
                    ))}
                  </select>
                </label>

                {field('tema', 'Tema', 'text', 255)}
                {field('nats_alkitab', 'Nats Alkitab', 'text', 100)}
                {field('pembicara', 'Pembicara', 'text', 100)}
                {field('lokasi', 'Lokasi', 'text', 150)}

                <label style={{ ...styles.label, gridColumn: '1 / -1' }}>
                  Deskripsi
                  <textarea
                    rows={3}
                    value={form.deskripsi}
                    onChange={event =>
                      changeField('deskripsi', event.target.value)
                    }
                    style={styles.input}
                  />
                </label>
              </div>
            )}

            <div style={{ ...styles.row, justifyContent: 'flex-end' }}>
              <button type="button" onClick={closeModal} style={styles.secondary}>
                Batal
              </button>
              <button type="submit" style={styles.primary}>
                {busy ? 'Menyimpan…' : 'Simpan kegiatan'}
              </button>
            </div>
          </fieldset>
        </form>
      </section>
    </div>
  );
}

export default function JadwalKegiatan() {
  const state = useJadwalKegiatan();

  const {
    month, moveMonth, thisMonth,
    groups, visibleItems, items,
    communities, communityFilter, setCommunityFilter, search, setSearch,
    loading, summary, summaryLoading,
    detail, detailLoading, openDetail, closeDetail,
    modal, openForm, busy, error, notice, remove, reload,
  } = state;

  const period = month.toLocaleDateString('id-ID', {
    month: 'long',
    year: 'numeric',
  });

  return (
    <div style={styles.page}>
      <style>{`
        .activity-page button:focus-visible,
        .activity-page a:focus-visible,
        .activity-page input:focus-visible,
        .activity-page select:focus-visible,
        .activity-page textarea:focus-visible {
          outline: 3px solid #C18B16;
          outline-offset: 3px;
        }
        .activity-page button:disabled {
          opacity: .55;
          cursor: not-allowed;
        }
      `}</style>

      <div className="activity-page" style={styles.page}>
        <div style={styles.header}>
          <div>
            <h1 style={styles.title}>Jadwal kegiatan</h1>
            <p style={styles.small}>
              Kelola kegiatan, petugas pelayanan, dan kebutuhan evaluasi.
            </p>
          </div>

          <div style={styles.row}>
            <Link to="/impor-jadwal" style={styles.secondary}>
              <Icon src={uploadIcon} /> Impor Excel
            </Link>
            <button
              type="button"
              style={styles.primary}
              disabled={busy}
              onClick={() => openForm('create')}
            >
              <Icon src={plusIcon} /> Tambah kegiatan
            </button>
          </div>
        </div>

        <div style={styles.toolbar}>
          <div style={styles.row}>
            <button
              type="button"
              aria-label="Bulan sebelumnya"
              style={styles.secondary}
              disabled={busy}
              onClick={() => moveMonth(-1)}
            >
              <Icon src={leftIcon} />
            </button>

            <strong style={{ textTransform: 'capitalize' }}>{period}</strong>

            <button
              type="button"
              aria-label="Bulan berikutnya"
              style={styles.secondary}
              disabled={busy}
              onClick={() => moveMonth(1)}
            >
              <Icon src={rightIcon} />
            </button>

            <button
              type="button"
              style={styles.secondary}
              disabled={busy}
              onClick={thisMonth}
            >
              Bulan ini
            </button>
          </div>

          <button
            type="button"
            style={styles.secondary}
            disabled={busy || loading || summaryLoading}
            onClick={reload}
          >
            <Icon src={refreshIcon} /> Muat ulang
          </button>
        </div>

        <div style={styles.row}>
          <input
            type="search"
            aria-label="Cari kegiatan"
            placeholder="Cari kegiatan, tema, pembicara, atau lokasi…"
            value={search}
            onChange={event => setSearch(event.target.value)}
            style={{ ...styles.input, flex: '1 1 260px' }}
          />

          <select
            aria-label="Filter komunitas"
            value={communityFilter}
            onChange={event => setCommunityFilter(event.target.value)}
            style={styles.input}
          >
            <option value="">Semua komunitas</option>
            <option value="umum">Umum</option>
            {communities.map(item => (
              <option key={item.id} value={String(item.id)}>
                {item.nama}
              </option>
            ))}
          </select>

          <span style={styles.small}>
            {visibleItems.length} dari {items.length} kegiatan
          </span>
        </div>

        {error && <div role="alert" style={styles.error}>{error}</div>}
        {notice && <div role="status" style={styles.success}>{notice}</div>}

        <div style={styles.columns}>
          <section style={{ ...styles.card, flex: '2 1 450px', minWidth: 0 }}>
            {loading ? (
              <div style={styles.empty}>Memuat kegiatan…</div>
            ) : groups.length === 0 ? (
              <div style={styles.empty}>
                Belum ada kegiatan sesuai periode dan filter.
              </div>
            ) : groups.map(([date, activities]) => (
              <div key={date}>
                <h2 style={styles.dateHeading}>
                  <Icon src={calendarIcon} />
                  {readableDate(date)}
                </h2>

                {activities.map(item => {
                  const count = summary[item.id]?.people;

                  return (
                    <button
                      key={item.id}
                      type="button"
                      disabled={busy}
                      aria-pressed={detail?.id === item.id}
                      onClick={() => openDetail(item)}
                      style={{
                        ...styles.activity,
                        background: detail?.id === item.id ? '#F0F9F9' : '#fff',
                        borderLeftColor: detail?.id === item.id
                          ? '#00777C'
                          : 'transparent',
                      }}
                    >
                      <span style={styles.time}>
                        {item.waktu_mulai}
                        {item.waktu_selesai && (
                          <span style={styles.small}>
                            – {item.waktu_selesai}
                          </span>
                        )}
                      </span>

                      <span style={{ flex: 1, minWidth: 0 }}>
                        <strong>{item.nama_kegiatan}</strong>
                        <span style={{ ...styles.small, display: 'block' }}>
                          {item.nama_komunitas || 'Umum'}
                          {item.lokasi ? ` · ${item.lokasi}` : ''}
                        </span>

                        <span style={{ ...styles.row, marginTop: 8 }}>
                          <span style={{
                            ...styles.badge,
                            background: count > 0 ? '#FFF4DE' : '#EEF3F3',
                          }}>
                            <Icon src={peopleIcon} />
                            {count === undefined
                              ? (summaryLoading ? 'Memuat petugas…' : 'Petugas belum dimuat')
                              : count === 0
                                ? 'Belum ada petugas'
                                : `${count} orang melayani`}
                          </span>

                          {item.perlu_evaluasi && (
                            <span style={styles.badge}>Perlu evaluasi</span>
                          )}
                        </span>
                      </span>
                    </button>
                  );
                })}
              </div>
            ))}
          </section>

          {(detail || detailLoading) && (
            <aside style={{
              ...styles.card,
              flex: '1 1 320px',
              minWidth: 0,
              padding: 20,
              display: 'flex',
              flexDirection: 'column',
              gap: 16,
            }}>
              <div style={styles.header}>
                <h2 style={styles.sectionTitle}>Detail kegiatan</h2>
                <button
                  type="button"
                  aria-label="Tutup detail"
                  style={styles.secondary}
                  disabled={busy}
                  onClick={closeDetail}
                >
                  <Icon src={closeIcon} />
                </button>
              </div>

              {detailLoading ? (
                <div style={styles.empty}>Memuat detail…</div>
              ) : detail && (
                <>
                  <strong style={{ fontSize: 18 }}>{detail.nama_kegiatan}</strong>

                  {[
                    ['Tanggal', readableDate(detail.tanggal)],
                    ['Jam', `${detail.waktu_mulai}${detail.waktu_selesai ? `–${detail.waktu_selesai}` : ''}`],
                    ['Komunitas', detail.nama_komunitas || 'Umum'],
                    ['Tema', detail.tema],
                    ['Nats Alkitab', detail.nats_alkitab],
                    ['Pembicara', detail.pembicara],
                    ['Lokasi', detail.lokasi],
                    ['Deskripsi', detail.deskripsi],
                    ['Evaluasi', detail.perlu_evaluasi ? 'Diperlukan' : 'Tidak diperlukan'],
                  ].map(([label, value]) => (
                    <div key={label}>
                      <div style={styles.small}>{label}</div>
                      <div style={{ marginTop: 4, whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>
                        {value || '—'}
                      </div>
                    </div>
                  ))}

                  <Link
                    to={`/penugasan-pelayan?tanggal=${detail.tanggal}&kegiatan=${detail.id}`}
                    style={styles.primary}
                  >
                    <Icon src={peopleIcon} /> Atur petugas
                  </Link>

                  <div style={styles.row}>
                    <button
                      type="button"
                      style={styles.secondary}
                      disabled={busy}
                      onClick={() => openForm('edit')}
                    >
                      <Icon src={editIcon} /> Edit
                    </button>

                    <button
                      type="button"
                      style={styles.danger}
                      disabled={busy}
                      onClick={remove}
                    >
                      Hapus kegiatan
                    </button>
                  </div>
                </>
              )}
            </aside>
          )}
        </div>

        {modal && <ActivityModal state={state} />}
      </div>
    </div>
  );
}

const button = {
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 8,
  padding: '9px 13px',
  borderRadius: 8,
  fontFamily: 'inherit',
  fontWeight: 600,
  fontSize: 13,
  cursor: 'pointer',
  textDecoration: 'none',
};

const styles = {
  page: { display: 'flex', flexDirection: 'column', gap: 20 },
  title: { margin: 0, fontSize: 28, color: '#0D1F21' },
  sectionTitle: { margin: 0, fontSize: 17, color: '#0D1F21' },
  small: { color: '#66787A', fontSize: 12, lineHeight: 1.6 },
  header: {
    display: 'flex', justifyContent: 'space-between',
    alignItems: 'center', flexWrap: 'wrap', gap: 12,
  },
  row: { display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 10 },
  toolbar: {
    padding: 14, background: '#fff', border: '1px solid #E1E8E8',
    borderRadius: 12, display: 'flex', justifyContent: 'space-between',
    flexWrap: 'wrap', gap: 12, alignItems: 'center',
  },
  columns: { display: 'flex', flexWrap: 'wrap', gap: 20, alignItems: 'flex-start' },
  card: {
    background: '#fff', border: '1px solid #E1E8E8',
    borderRadius: 12, overflow: 'hidden', boxSizing: 'border-box',
  },
  dateHeading: {
    margin: 0, padding: '14px 18px', display: 'flex', alignItems: 'center',
    gap: 10, fontSize: 13, color: '#075E62', background: '#F2F8F8',
  },
  activity: {
    display: 'flex', alignItems: 'flex-start', gap: 16,
    width: '100%', padding: '18px 16px', textAlign: 'left',
    border: 0, borderLeft: '3px solid transparent',
    borderBottom: '1px solid #E1E8E8', fontFamily: 'inherit',
    color: '#0D1F21', fontSize: 14, cursor: 'pointer',
  },
  time: { display: 'flex', flexDirection: 'column', minWidth: 55, fontWeight: 600 },
  badge: {
    display: 'inline-flex', alignItems: 'center', gap: 6, padding: '3px 8px',
    borderRadius: 6, background: '#E0F3F2', color: '#445759', fontSize: 11,
  },
  primary: { ...button, border: '1px solid #00777C', background: '#00777C', color: '#fff' },
  secondary: { ...button, border: '1px solid #CBD6D6', background: '#fff', color: '#0D1F21' },
  danger: { ...button, border: '1px solid #F3B9B9', background: '#FFF3F3', color: '#B42318' },
  input: {
    padding: '10px 12px', border: '1px solid #CBD6D6', borderRadius: 8,
    fontFamily: 'inherit', fontSize: 14, minWidth: 0, boxSizing: 'border-box',
    background: '#fff', color: '#0D1F21',
  },
  empty: { padding: '50px 20px', textAlign: 'center', color: '#66787A', fontSize: 13 },
  error: { padding: 12, background: '#FFF0F0', color: '#B42318', borderRadius: 8, fontSize: 13 },
  success: { padding: 12, background: '#E0F3E8', color: '#09703E', borderRadius: 8, fontSize: 13 },
  overlay: {
    position: 'fixed', inset: 0, background: 'rgba(13,31,33,.6)',
    display: 'flex', justifyContent: 'center', alignItems: 'center',
    padding: 20, zIndex: 9999,
  },
  modal: {
    width: 650, maxWidth: '100%', maxHeight: '90vh', overflowY: 'auto',
    background: '#fff', padding: 24, borderRadius: 14, boxSizing: 'border-box',
    display: 'flex', flexDirection: 'column', gap: 18,
  },
  fieldset: {
    border: 0, margin: 0, padding: 0, minWidth: 0,
    display: 'flex', flexDirection: 'column', gap: 18,
  },
  label: { display: 'flex', flexDirection: 'column', gap: 7, fontSize: 13, color: '#445759', fontWeight: 600 },
  checkbox: { display: 'flex', alignItems: 'center', gap: 8, fontSize: 13 },
  grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 16 },
  fieldError: { fontSize: 12, color: '#B42318' },
};