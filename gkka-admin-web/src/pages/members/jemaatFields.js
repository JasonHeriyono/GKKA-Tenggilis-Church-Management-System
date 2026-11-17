export const roles = ['Jemaat', 'Majelis', 'Super Admin'];

export const identityFields = [
  { key: 'nik', label: 'NIK', maxLength: 16, pattern: '[0-9]{16}' },
  { key: 'nama_panggilan', label: 'Nama panggilan', maxLength: 50 },
  { key: 'jenis_kelamin', label: 'Jenis kelamin', options: ['L', 'P'] },
  { key: 'tempat_lahir', label: 'Tempat lahir', maxLength: 100 },
  { key: 'tanggal_lahir', label: 'Tanggal lahir', type: 'date' },
  { key: 'alamat_ktp', label: 'Alamat KTP', type: 'textarea', maxLength: 500 },
  { key: 'rt_rw', label: 'RT/RW', maxLength: 10 },
  { key: 'kelurahan', label: 'Kelurahan', maxLength: 100 },
  { key: 'kecamatan', label: 'Kecamatan', maxLength: 100 },
  { key: 'kota', label: 'Kota', maxLength: 100 },
  { key: 'provinsi', label: 'Provinsi', maxLength: 100 },
  {
    key: 'alamat_domisili',
    label: 'Alamat domisili',
    type: 'textarea',
    maxLength: 500,
  },
  { key: 'kewarganegaraan', label: 'Kewarganegaraan', options: ['WNI', 'WNA'] },
  {
    key: 'status_pernikahan',
    label: 'Status pernikahan',
    options: ['BELUM_KAWIN', 'KAWIN', 'CERAI_HIDUP', 'CERAI_MATI'],
  },
  { key: 'tanggal_pernikahan', label: 'Tanggal pernikahan', type: 'date' },
  {
    key: 'pendidikan_terakhir',
    label: 'Pendidikan terakhir',
    options: ['SD', 'SMP', 'SMA', 'D3', 'S1', 'S2', 'S3'],
  },
  { key: 'pekerjaan', label: 'Pekerjaan', maxLength: 100 },
  { key: 'nama_ayah', label: 'Nama ayah', maxLength: 100 },
  { key: 'nama_ibu', label: 'Nama ibu', maxLength: 100 },
  {
    key: 'status_gereja',
    label: 'Status gereja',
    options: ['JEMAAT_GKKA', 'SIMPATISAN', 'TAMU'],
  },
  { key: 'sudah_baptis', label: 'Sudah baptis', type: 'checkbox' },
  { key: 'tanggal_baptis', label: 'Tanggal baptis', type: 'date' },
  { key: 'gereja_baptis', label: 'Gereja baptis', maxLength: 150 },
  { key: 'sudah_sidi', label: 'Sudah sidi', type: 'checkbox' },
  { key: 'tanggal_sidi', label: 'Tanggal sidi', type: 'date' },
  { key: 'gereja_sidi', label: 'Gereja sidi', maxLength: 150 },
  { key: 'gereja_asal', label: 'Gereja asal', maxLength: 150 },
  { key: 'tanggal_bergabung', label: 'Tanggal bergabung', type: 'date' },
  { key: 'tanggal_keluar', label: 'Tanggal keluar', type: 'date' },
  {
    key: 'alasan_keluar',
    label: 'Alasan keluar',
    options: ['PINDAH', 'ATESTASI_KELUAR', 'MENINGGAL'],
  },
];

export const formatLabel = value =>
  String(value || '').replaceAll('_', ' ');

export function createForm(user = null) {
  const identitas = Object.fromEntries(
    identityFields.map(field => [
      field.key,
      field.type === 'checkbox'
        ? Boolean(user?.identitas?.[field.key])
        : (user?.identitas?.[field.key] ?? ''),
    ]),
  );

  return {
    kode_jemaat: user?.kode_jemaat ?? '',
    nama: user?.nama ?? '',
    email: user?.email ?? '',
    no_telepon: user?.no_telepon ?? '',
    role: user?.role ?? 'Jemaat',
    password: '',
    identitas,
  };
}