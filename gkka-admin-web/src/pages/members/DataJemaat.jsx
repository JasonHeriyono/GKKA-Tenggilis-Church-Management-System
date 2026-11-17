import useDataJemaat from './useDataJemaat';
import { identityFields, roles, formatLabel } from './jemaatFields';
import { useState } from 'react';

export default function DataJemaat() {
  const state = useDataJemaat();

  const {
    jemaat, meta, isLoading, errorMsg, notice,
    searchQuery, setSearchQuery, filters, applyFilter, submitSearch,
    page, setPage,
    selectedUser, detailLoading, openDetail, closeDrawer,
    modal, openModal, closeModal, formData, changeField, changeIdentity,
    formError, fieldErrors, isSaving, submitForm,
    isBusy, isSelf, toggleActive, deleteUser,
  } = state;

  const locked = isBusy || isSaving;
  const editingIdentity = modal === 'create' || modal === 'edit';
  const [showAdditionalIdentity, setShowAdditionalIdentity] = useState(false);

  const basicIdentityKeys = [
    'jenis_kelamin',
    'tanggal_lahir',
    'status_gereja',
  ];

const visibleIdentityFields =
  modal === 'edit' || showAdditionalIdentity
    ? identityFields
    : identityFields.filter(field => basicIdentityKeys.includes(field.key));

  const titles = {
    create: 'Tambah jemaat',
    edit: 'Edit data jemaat',
    role: 'Ubah role',
    password: 'Atur password',
  };

  function renderIdentityInput(field) {
    const value = formData.identitas[field.key];

    const common = {
      id: `identitas-${field.key}`,
      name: field.key,
      style: styles.input,
      value,
      onChange: event => changeIdentity(field.key, event.target.value),
    };

    if (field.type === 'checkbox') {
      return (
        <input
          id={common.id}
          type="checkbox"
          checked={value}
          onChange={event => changeIdentity(field.key, event.target.checked)}
        />
      );
    }

    if (field.options) {
      return (
        <select {...common}>
          <option value="">Belum diisi</option>
          {field.options.map(option => (
            <option key={option} value={option}>
              {formatLabel(option)}
            </option>
          ))}
        </select>
      );
    }

    if (field.type === 'textarea') {
      return (
        <textarea
          {...common}
          rows={3}
          maxLength={field.maxLength}
        />
      );
    }

    return (
      <input
        {...common}
        type={field.type || 'text'}
        maxLength={field.maxLength}
        pattern={field.pattern}
        inputMode={field.key === 'nik' ? 'numeric' : undefined}
      />
    );
  }

  return (
    <div style={styles.page}>
      <div style={styles.header}>
        <div>
          <h1 style={styles.title}>Data jemaat</h1>
          <p style={styles.subtitle}>
            {meta?.total ?? 0} jemaat sesuai filter.
          </p>
        </div>

        <button
          type="button"
          style={styles.primary}
          disabled={locked}
         onClick={() => {
            setShowAdditionalIdentity(false);
            openModal('create');
          }}
        >
          + Tambah jemaat
        </button>
      </div>

      {errorMsg && <div role="alert" style={styles.error}>{errorMsg}</div>}
      {notice && <div role="status" style={styles.success}>{notice}</div>}

      <div style={styles.columns}>
        <section style={styles.tableCard}>
          <div style={styles.toolbar}>
            <form onSubmit={submitSearch} style={styles.searchForm}>
              <input
                aria-label="Cari jemaat"
                placeholder="Nama, kode, email, atau no. HP"
                value={searchQuery}
                onChange={event => setSearchQuery(event.target.value)}
                style={styles.input}
              />
              <button type="submit" style={styles.secondary}>
                Cari
              </button>
            </form>

            <select
              aria-label="Filter role"
              value={filters.role}
              onChange={event => applyFilter('role', event.target.value)}
              style={styles.input}
            >
              <option value="">Semua role</option>
              {roles.map(role => (
                <option key={role} value={role}>{role}</option>
              ))}
            </select>

            <select
              aria-label="Filter verifikasi"
              value={filters.status_verifikasi}
              onChange={event =>
                applyFilter('status_verifikasi', event.target.value)
              }
              style={styles.input}
            >
              <option value="">Semua verifikasi</option>
              {[
                'BELUM_TERVERIFIKASI',
                'MENUNGGU_VERIFIKASI',
                'TERVERIFIKASI',
              ].map(status => (
                <option key={status} value={status}>
                  {formatLabel(status)}
                </option>
              ))}
            </select>
          </div>

          <div style={{ overflowX: 'auto' }}>
            <table style={styles.table}>
              <thead>
                <tr>
                  {['Kode', 'Nama', 'Kontak', 'Role', 'Status', 'Detail'].map(
                    title => <th key={title} style={styles.th}>{title}</th>,
                  )}
                </tr>
              </thead>

              <tbody>
                {isLoading ? (
                  <tr>
                    <td colSpan={6} style={styles.empty}>Memuat data…</td>
                  </tr>
                ) : jemaat.length === 0 ? (
                  <tr>
                    <td colSpan={6} style={styles.empty}>Data tidak ditemukan.</td>
                  </tr>
                ) : jemaat.map(user => (
                  <tr
                    key={user.id}
                    style={{
                      background:
                        selectedUser?.id === user.id ? '#F0F9F9' : '#fff',
                    }}
                  >
                    <td style={styles.td}>{user.kode_jemaat || '—'}</td>
                    <td style={styles.td}>
                      <strong>{user.nama}</strong>
                      <div style={styles.small}>
                        {formatLabel(user.status_verifikasi)}
                      </div>
                    </td>
                    <td style={styles.td}>
                      <div>{user.email || '—'}</div>
                      <div style={styles.small}>{user.no_telepon || '—'}</div>
                    </td>
                    <td style={styles.td}>{user.role}</td>
                    <td style={styles.td}>
                      <span
                        style={{
                          ...styles.badge,
                          background: user.is_aktif ? '#E0F3E8' : '#FFE5E5',
                          color: user.is_aktif ? '#09703E' : '#B42318',
                        }}
                      >
                        {user.is_aktif ? 'Aktif' : 'Nonaktif'}
                      </span>
                    </td>
                    <td style={styles.td}>
                      <button
                        type="button"
                        style={styles.secondary}
                        disabled={locked || detailLoading}
                        onClick={() => openDetail(user)}
                      >
                        Detail
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div style={styles.pagination}>
            <span style={styles.small}>
              Halaman {meta?.page ?? page} dari {meta?.total_pages || 1}
              {' · '}{meta?.total ?? 0} hasil
            </span>
            <div style={styles.actions}>
              <button
                type="button"
                style={styles.secondary}
                disabled={isLoading || locked || page <= 1}
                onClick={() => {
                  closeDrawer();
                  setPage(value => value - 1);
                }}
              >
                Sebelumnya
              </button>
              <button
                type="button"
                style={styles.secondary}
                disabled={
                  isLoading || locked || !meta ||
                  page >= meta.total_pages
                }
                onClick={() => {
                  closeDrawer();
                  setPage(value => value + 1);
                }}
              >
                Berikutnya
              </button>
            </div>
          </div>
        </section>

        {(selectedUser || detailLoading) && (
          <aside style={styles.drawer}>
            <div style={styles.drawerHeader}>
              <strong>{selectedUser?.nama || 'Memuat detail…'}</strong>
              <button
                type="button"
                aria-label="Tutup detail"
                style={styles.secondary}
                disabled={locked}
                onClick={closeDrawer}
              >
                ×
              </button>
            </div>

            {selectedUser && (
              <>
                <div style={styles.drawerBody}>
                  {[
                    ['Kode jemaat', selectedUser.kode_jemaat],
                    ['Role', selectedUser.role],
                    ['Verifikasi', formatLabel(selectedUser.status_verifikasi)],
                    ['Email', selectedUser.email],
                    ['No. HP', selectedUser.no_telepon],
                    ['Status akun', selectedUser.is_aktif ? 'Aktif' : 'Nonaktif'],
                    [
                      'Password',
                      selectedUser.account_activated ? 'Sudah diatur' : 'Belum diatur',
                    ],
                    ...identityFields.map(field => {
                      const value = selectedUser.identitas?.[field.key];
                      return [
                        field.label,
                        field.type === 'checkbox'
                          ? (value ? 'Ya' : 'Tidak')
                          : formatLabel(value),
                      ];
                    }),
                  ].map(([label, value]) => (
                    <div key={label}>
                      <div style={styles.small}>{label}</div>
                      <div style={styles.detailValue}>{value || '—'}</div>
                    </div>
                  ))}
                </div>

                <div style={styles.drawerFooter}>
                  <button
                    type="button"
                    style={styles.primary}
                    disabled={locked}
                    onClick={() => openModal('edit')}
                  >
                    Edit data
                  </button>

                  <button
                    type="button"
                    style={styles.secondary}
                    disabled={locked || isSelf}
                    onClick={() => openModal('role')}
                  >
                    Ubah role
                  </button>

                  <button
                    type="button"
                    style={styles.secondary}
                    disabled={locked}
                    onClick={() => openModal('password')}
                  >
                    Atur password
                  </button>

                  <button
                    type="button"
                    style={styles.secondary}
                    disabled={locked || (isSelf && selectedUser.is_aktif)}
                    onClick={toggleActive}
                  >
                    {selectedUser.is_aktif ? 'Nonaktifkan' : 'Aktifkan'}
                  </button>

                  <button
                    type="button"
                    style={styles.danger}
                    disabled={locked || isSelf}
                    onClick={deleteUser}
                  >
                    Hapus jemaat
                  </button>
                </div>
              </>
            )}
          </aside>
        )}
      </div>

      {modal && (
        <div style={styles.overlay}>
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="jemaat-modal-title"
            style={styles.modal}
          >
            <h2 id="jemaat-modal-title" style={styles.modalTitle}>
              {titles[modal]}
            </h2>

            {formError && (
              <div role="alert" style={styles.error}>{formError}</div>
            )}

            {Object.keys(fieldErrors).length > 0 && (
              <ul style={styles.error}>
                {Object.entries(fieldErrors).map(([field, message]) => (
                  <li key={field}>{field}: {message}</li>
                ))}
              </ul>
            )}

            <form onSubmit={submitForm}>
              <fieldset disabled={isSaving} style={styles.fieldset}>
                {editingIdentity && (
                  <>
                    <div style={styles.grid}>
                      {[
                        ['kode_jemaat', 'Kode jemaat', 'text', 20],
                        ['nama', 'Nama lengkap', 'text', 100],
                        ['email', 'Email', 'email', 100],
                        ['no_telepon', 'No. HP', 'tel', 20],
                      ].map(([key, label, type, maxLength]) => (
                        <label key={key} style={styles.label}>
                          {label}
                          <input
                            name={key}
                            type={type}
                            value={formData[key]}
                            required={key === 'nama'}
                            maxLength={maxLength}
                            placeholder={
                              key === 'kode_jemaat'
                                ? (modal === 'create' ? 'Otomatis jika dikosongkan' : 'Kode jemaat')
                                : key === 'email'
                                  ? 'Opsional'
                                  : key === 'no_telepon'
                                    ? 'Opsional'
                                    : ''
                            }
                            onChange={event => changeField(key, event.target.value)}
                            style={styles.input}
                          />
                        </label>
                      ))}
                    </div>

                    <h3 style={styles.sectionTitle}>Identitas jemaat</h3>

                    <div style={styles.identityHeading}>
                    <h3 style={styles.sectionTitle}>
                      {modal === 'create' ? 'Data dasar' : 'Identitas jemaat'}
                    </h3>

                    {modal === 'create' && (
                      <button
                        type="button"
                        style={styles.secondary}
                        aria-expanded={showAdditionalIdentity}
                        onClick={() => setShowAdditionalIdentity(value => !value)}
                      >
                        {showAdditionalIdentity
                          ? 'Sembunyikan data tambahan'
                          : '+ Lengkapi data tambahan'}
                      </button>
                    )}
                  </div>

                  {modal === 'create' && (
                    <p style={styles.formHint}>
                      Cukup isi nama untuk mendaftarkan jemaat.
                      Data lainnya dapat dilengkapi melalui Edit data.
                    </p>
                  )}

                  <div style={styles.grid}>
                    {visibleIdentityFields.map(field => (
                      <label
                        key={field.key}
                        htmlFor={`identitas-${field.key}`}
                        style={styles.label}
                      >
                        {field.label}
                        {renderIdentityInput(field)}
                      </label>
                    ))}
                  </div>
                  </>
                )}

                {(modal === 'create' || modal === 'role') && (
                  <label style={styles.label}>
                    Role
                    <select
                      value={formData.role}
                      onChange={event => changeField('role', event.target.value)}
                      style={styles.input}
                    >
                      {roles.map(role => (
                        <option key={role} value={role}>{role}</option>
                      ))}
                    </select>
                  </label>
                )}

                {(modal === 'create' || modal === 'password') && (
                  <label style={styles.label}>
                    {modal === 'create'
                      ? 'Password awal (opsional)'
                      : 'Password baru'}
                    <input
                      type="password"
                      autoComplete="new-password"
                      value={formData.password}
                      required={modal === 'password'}
                      minLength={8}
                      maxLength={72}
                      onChange={event => changeField('password', event.target.value)}
                      style={styles.input}
                    />
                  </label>
                )}

                <div style={styles.modalActions}>
                  <button
                    type="button"
                    style={styles.secondary}
                    onClick={closeModal}
                  >
                    Batal
                  </button>
                  <button type="submit" style={styles.primary}>
                    {isSaving ? 'Menyimpan…' : 'Simpan'}
                  </button>
                </div>
              </fieldset>
            </form>
          </section>
        </div>
      )}
    </div>
  );
}

const button = {
  padding: '9px 12px',
  borderRadius: 8,
  fontSize: 13,
  fontWeight: 600,
  fontFamily: 'inherit',
  cursor: 'pointer',
};

const styles = {
  page: { display: 'flex', flexDirection: 'column', gap: 20 },
  header: {
    display: 'flex', justifyContent: 'space-between',
    alignItems: 'center', gap: 16,
  },
  title: { margin: 0, fontSize: 28, color: '#0D1F21' },
  subtitle: { margin: '6px 0 0', color: '#66787A', fontSize: 14 },
  primary: {
    ...button, border: '1px solid #00777C',
    background: '#00777C', color: '#fff',
  },
  secondary: {
    ...button, border: '1px solid #CBD6D6',
    background: '#fff', color: '#0D1F21',
  },
  danger: {
    ...button, border: '1px solid #F3B9B9',
    background: '#FFF3F3', color: '#B42318',
  },
  columns: { display: 'flex', alignItems: 'flex-start', gap: 20 },
  tableCard: {
    flex: 1, minWidth: 0, background: '#fff',
    border: '1px solid #E1E8E8', borderRadius: 12, overflow: 'hidden',
  },
  toolbar: {
    padding: 16, display: 'flex', gap: 10,
    flexWrap: 'wrap', borderBottom: '1px solid #E1E8E8',
  },
  searchForm: { display: 'flex', gap: 8, flex: '1 1 280px' },
  input: {
    width: '100%', boxSizing: 'border-box', minWidth: 0,
    padding: '10px 12px', border: '1px solid #CBD6D6',
    borderRadius: 8, background: '#fff', color: '#0D1F21',
    fontSize: 14, fontFamily: 'inherit',
  },
  table: { width: '100%', borderCollapse: 'collapse', textAlign: 'left' },
  th: {
    padding: '12px 16px', background: '#F7FAFA',
    color: '#66787A', fontSize: 12, whiteSpace: 'nowrap',
  },
  td: {
    padding: '12px 16px', borderBottom: '1px solid #E1E8E8',
    color: '#0D1F21', fontSize: 13, overflowWrap: 'anywhere',
  },
  empty: { padding: 60, textAlign: 'center', color: '#66787A' },
  small: { fontSize: 12, color: '#66787A', marginTop: 4 },
  badge: { padding: '3px 8px', borderRadius: 5, fontSize: 11 },
  pagination: {
    padding: 16, display: 'flex',
    justifyContent: 'space-between', alignItems: 'center', gap: 12,
  },
  actions: { display: 'flex', gap: 8 },
  drawer: {
    width: 320, flexShrink: 0, background: '#fff',
    border: '1px solid #E1E8E8', borderRadius: 12, overflow: 'hidden',
  },
  drawerHeader: {
    padding: 16, background: '#E0F3F2', display: 'flex',
    justifyContent: 'space-between', alignItems: 'center', gap: 12,
  },
  drawerBody: {
    padding: 20, display: 'grid', gap: 14,
    maxHeight: '60vh', overflowY: 'auto',
  },
  detailValue: {
    marginTop: 4, fontSize: 14, color: '#0D1F21',
    whiteSpace: 'pre-wrap', overflowWrap: 'anywhere',
  },
  drawerFooter: {
    padding: 16, borderTop: '1px solid #E1E8E8',
    display: 'flex', flexWrap: 'wrap', gap: 8,
  },
  overlay: {
    position: 'fixed', inset: 0, background: 'rgba(13,31,33,.6)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    padding: 20, zIndex: 9999,
  },
  modal: {
    width: 760, maxWidth: '100%', maxHeight: '90vh',
    overflowY: 'auto', background: '#fff', padding: 24,
    borderRadius: 12, boxSizing: 'border-box',
  },
  modalTitle: { margin: '0 0 20px', color: '#0D1F21', fontSize: 20 },
  fieldset: {
    border: 0, padding: 0, margin: 0, minWidth: 0,
    display: 'flex', flexDirection: 'column', gap: 18,
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
    gap: 16,
  },
  label: {
    display: 'flex', flexDirection: 'column', gap: 7,
    fontSize: 13, color: '#445759', fontWeight: 600,
  },
  sectionTitle: { margin: 0, color: '#00777C', fontSize: 16 },
  modalActions: { display: 'flex', justifyContent: 'flex-end', gap: 10 },
  error: {
    background: '#FFF0F0', color: '#B42318',
    padding: '12px 20px', borderRadius: 8, fontSize: 13,
  },
  success: {
    background: '#E0F3E8', color: '#09703E',
    padding: '12px 16px', borderRadius: 8, fontSize: 13,
  },

  identityHeading: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    flexWrap: 'wrap',
    gap: 12,
  },
  formHint: {
    margin: 0,
    color: '#66787A',
    fontSize: 13,
    lineHeight: 1.6,
  },
};