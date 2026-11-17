import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import apiClient from '../../services/api'; // Pastikan path ini benar mengarah ke file api.js kamu
import { saveSession } from '../../services/session';

// Import aset SVG sesuai dengan desain login_2.html
import churchIcon from '../../assets/icon-church.svg';
import userIcon from '../../assets/icon-user.svg';
import lockIcon from '../../assets/icon-lock.svg';
import eyeIcon from '../../assets/icon-eye.svg';
import checkIcon from '../../assets/icon-check.svg';

export default function Login() {
  const navigate = useNavigate();

  const [formData, setFormData] = useState({
    identifier: '',
    password: '',
  });

  const [showPassword, setShowPassword] = useState(false);
  const [rememberMe, setRememberMe] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  const handleChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    });
  };

  const handleSubmit = async (e) => {
  e.preventDefault();
  if (isLoading) return;
  setIsLoading(true);
  setErrorMsg('');
  try {
    const response = await apiClient.post('/auth/login', {
      identifier: formData.identifier.trim(),
      password: formData.password,
    });
    const data = response.data.data;
    if (!['Super Admin', 'Majelis'].includes(data?.user?.role)) {
      // Akun valid, tetapi bukan pengguna web admin. Cabut sesi yang baru diterbitkan.
      if (data?.refresh_token) {
        try { await apiClient.post('/auth/logout', { refresh_token: data.refresh_token }); }
        catch { /* Sesi ini tidak disimpan di browser. */ }
      }
      setErrorMsg('Web admin hanya untuk Super Admin dan Majelis. Jemaat menggunakan aplikasi jemaat.');
      return;
    }
    saveSession(data, rememberMe);
    navigate(data.user.role === 'Majelis' ? '/laporan-majelis' : '/', { replace: true });
  } catch (err) {
    setErrorMsg(err.response?.data?.message || err.message || 'Tidak dapat terhubung ke server.');
  } finally {
    setIsLoading(false);
  }
};

   return (
    <div style={styles.pageWrapper}>
      {/* Kolom Kiri: Sidebar Informasi */}
      <div style={styles.leftColumn}>
        {/* Brand Header */}
        <div style={styles.brandRow}>
          <div style={styles.logoBadge}>
            <img src={churchIcon} style={styles.iconChurch} alt="Gereja" />
          </div>
          <div style={styles.brandName}>GKKA Tenggilis</div>
        </div>

        {/* Headline Tengah */}
        <div style={styles.middleTextWrapper}>
          <div style={styles.headlineText}>
            Satu tempat untuk jadwal, absensi, dan evaluasi ibadah.
          </div>
          <div style={styles.descriptionText}>
            Sekretariat mengimpor jadwal dari Excel, usher mencatat kehadiran dari ponsel, dan majelis membaca hasil evaluasi setiap minggu.
          </div>
        </div>

        {/* Kartu Statistik Bawah */}
        <div style={styles.statsContainer}>
          <div style={styles.statBox}>
            <div style={styles.statValue}>412</div>
            <div style={styles.statLabel}>jemaat</div>
          </div>
          <div style={styles.statBox}>
            <div style={styles.statValue}>28</div>
            <div style={styles.statLabel}>usher</div>
          </div>
          <div style={styles.statBox}>
            <div style={styles.statValue}>4</div>
            <div style={styles.statLabel}>ibadah/minggu</div>
          </div>
        </div>
      </div>

      {/* Kolom Kanan: Card Form Login */}
      <div style={styles.rightColumn}>
        <div style={styles.formCard}>
          <div style={styles.cardTitle}>Masuk</div>
          <div style={styles.cardSubtitle}>Gunakan akun yang diberikan sekretariat gereja.</div>

          {errorMsg && <div style={styles.errorAlert}>{errorMsg}</div>}

          <form onSubmit={handleSubmit} style={styles.form}>
            {/* Input Identifier */}
            <div style={styles.fieldGroup}>
              <label style={styles.fieldLabel}>Email atau no. HP</label>
              <div style={styles.inputWrapper}>
                <img src={userIcon} style={styles.fieldIcon} alt="User" />
                <input
                  type="text"
                  name="identifier"
                  value={formData.identifier}
                  onChange={handleChange}
                  placeholder="Masukkan username atau no. HP"
                  style={styles.textInput}
                  required
                />
              </div>
            </div>

            {/* Input Password */}
            <div style={styles.fieldGroup}>
              <label style={styles.fieldLabel}>Kata sandi</label>
              <div style={styles.inputWrapper}>
                <img src={lockIcon} style={styles.fieldIcon} alt="Lock" />
                <input
                  type={showPassword ? 'text' : 'password'}
                  name="password"
                  value={formData.password}
                  onChange={handleChange}
                  placeholder="Masukkan kata sandi"
                  style={styles.textInput}
                  required
                />
                <img
                  src={eyeIcon}
                  onClick={() => setShowPassword(!showPassword)}
                  style={styles.eyeToggle}
                  alt="Toggle Visibility"
                />
              </div>
            </div>

            {/* Remember Me & Lupa Password */}
            <div style={styles.optionsRow}>
              <div
                style={styles.rememberWrapper}
                onClick={() => setRememberMe(!rememberMe)}
              >
                <div style={{ ...styles.checkbox, backgroundColor: rememberMe ? '#00777C' : 'transparent', borderColor: rememberMe ? '#00777C' : '#CBD6D6' }}>
                  {rememberMe && <img src={checkIcon} style={styles.checkIcon} alt="Checked" />}
                </div>
                <span style={styles.rememberLabel}>Ingat saya</span>
              </div>
              <span style={styles.forgotPassword}>Lupa kata sandi?</span>
            </div>

            {/* Tombol Masuk */}
            <button
              type="submit"
              disabled={isLoading}
              style={{ ...styles.submitButton, opacity: isLoading ? 0.7 : 1 }}
            >
              {isLoading ? 'Memeriksa...' : 'Masuk'}
            </button>
          </form>

          <div style={styles.footerNote}>
            Jemaat masuk lewat aplikasi di ponsel.
          </div>
        </div>
      </div>
    </div>
  );
}

const styles = {
  pageWrapper: {
    width: '100vw',
    minHeight: '100vh',
    backgroundColor: '#F4F7F7',
    display: 'flex',
    alignItems: 'stretch',
    justifyContent: 'flex-start',
  },
  leftColumn: {
    width: '640px',
    minHeight: '100vh',
    padding: '48px 56px',
    backgroundColor: '#0D2B2E',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    boxSizing: 'border-box',
  },
  brandRow: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '12px',
  },
  logoBadge: {
    width: '40px',
    height: '40px',
    backgroundColor: '#C9952B',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  iconChurch: {
    width: '22px',
    height: '22px',
    filter: 'brightness(0) invert(1)',
  },
  brandName: {
    color: '#E8F1F1',
    fontSize: '16px',
    fontWeight: '600',
    lineHeight: '24px',
  },
  middleTextWrapper: {
    display: 'flex',
    flexDirection: 'column',
    gap: '20px',
  },
  headlineText: {
    alignSelf: 'stretch',
    color: '#E8F1F1',
    fontSize: '30px',
    fontWeight: '800',
    lineHeight: '38px',
  },
  descriptionText: {
    alignSelf: 'stretch',
    color: '#93AFB1',
    fontSize: '14px',
    fontWeight: '400',
    lineHeight: '22px',
  },
  statsContainer: {
    alignSelf: 'stretch',
    display: 'inline-flex',
    gap: '12px',
  },
  statBox: {
    flex: '1 1 0',
    padding: '16px',
    backgroundColor: '#17403F',
    borderRadius: '12px',
    display: 'inline-flex',
    flexDirection: 'column',
    gap: '2px',
  },
  statValue: {
    color: '#E8F1F1',
    fontSize: '26px',
    fontWeight: '700',
    lineHeight: '32px',
  },
  statLabel: {
    color: '#93AFB1',
    fontSize: '12px',
    fontWeight: '500',
    lineHeight: '16px',
  },
  rightColumn: {
    flex: '1 1 0',
    display: 'inline-flex',
    flexDirection: 'column',
    justifyContent: 'center',
    alignItems: 'center',
    padding: '32px',
  },
  formCard: {
    width: '400px',
    padding: '32px',
    backgroundColor: '#FFFFFF',
    boxShadow: '0px 12px 32px -8px rgba(13, 31, 33, 0.12)',
    borderRadius: '16px',
    border: '1px solid #E1E8E8',
    display: 'flex',
    flexDirection: 'column',
    gap: '20px',
    boxSizing: 'border-box',
  },
  cardTitle: {
    color: '#0D1F21',
    fontSize: '20px',
    fontWeight: '700',
    lineHeight: '28px',
  },
  cardSubtitle: {
    alignSelf: 'stretch',
    color: '#445759',
    fontSize: '13px',
    fontWeight: '400',
    lineHeight: '20px',
  },
  errorAlert: {
    padding: '10px 12px',
    backgroundColor: '#FFE5E5',
    color: '#D8000C',
    borderRadius: '8px',
    fontSize: '13px',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
    width: '100%',
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
    lineHeight: '20px',
  },
  inputWrapper: {
    alignSelf: 'stretch',
    padding: '10px 12px',
    backgroundColor: '#FFFFFF',
    borderRadius: '8px',
    border: '1px solid #CBD6D6',
    display: 'inline-flex',
    alignItems: 'center',
    gap: '8px',
  },
  fieldIcon: {
    width: '18px',
    height: '18px',
  },
  textInput: {
    border: 'none',
    outline: 'none',
    flex: '1 1 0',
    color: '#0D1F21',
    fontSize: '14px',
    fontFamily: 'inherit',
    backgroundColor: 'transparent',
  },
  eyeToggle: {
    width: '18px',
    height: '18px',
    cursor: 'pointer',
    opacity: 0.7,
  },
  optionsRow: {
    alignSelf: 'stretch',
    display: 'inline-flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  rememberWrapper: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '8px',
    cursor: 'pointer',
  },
  checkbox: {
    width: '18px',
    height: '18px',
    borderRadius: '6px',
    border: '1px solid #CBD6D6',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  checkIcon: {
    width: '12px',
    height: '12px',
    filter: 'brightness(0) invert(1)',
  },
  rememberLabel: {
    color: '#0D1F21',
    fontSize: '13px',
    fontWeight: '400',
  },
  forgotPassword: {
    color: '#00777C',
    fontSize: '13px',
    fontWeight: '600',
    cursor: 'pointer',
  },
  submitButton: {
    alignSelf: 'stretch',
    padding: '10px 16px',
    backgroundColor: '#00777C',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
    color: '#FFFFFF',
    fontSize: '14px',
    fontWeight: '600',
    lineHeight: '22px',
    fontFamily: 'inherit',
  },
  footerNote: {
    alignSelf: 'stretch',
    textAlign: 'center',
    color: '#66787A',
    fontSize: '12px',
    fontWeight: '500',
    lineHeight: '16px',
  },
};