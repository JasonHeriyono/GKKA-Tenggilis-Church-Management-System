import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import Topbar from './Topbar';

export default function MainLayout() {
  return (
    <div style={styles.appContainer}>
      <Sidebar />

      <div style={styles.mainWrapper}>
        <Topbar />
        <main style={styles.contentArea}>
          {/* Pembungkus agar konten tidak gepeng atau terpotong paksa saat zoom tinggi */}
          <div style={styles.innerContent}>
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}

const styles = {
  appContainer: {
    display: 'flex',
    width: '100%',
    height: '100vh',
    overflow: 'hidden',
    backgroundColor: '#F4F7F7',
  },
  mainWrapper: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    height: '100vh',
    minWidth: 0,
    overflowY: 'auto',
    overflowX: 'auto', /* Mengizinkan scroll ke samping saat layar sempit/zoom tinggi */
  },
  contentArea: {
    flex: 1,
    padding: '24px 32px',
    boxSizing: 'border-box',
    display: 'flex',
    flexDirection: 'column',
  },
  innerContent: {
    width: '100%',
    minWidth: '960px', /* Menjaga batas layout grid & drawer tabel tidak remuk */
  },
};