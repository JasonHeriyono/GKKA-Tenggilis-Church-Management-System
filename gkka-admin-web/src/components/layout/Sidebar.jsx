import { useEffect, useState } from "react";
import { NavLink, useNavigate } from "react-router-dom";
import { getSession } from "../../services/session";
import api, { logout } from "../../services/api";

import churchIcon from "../../assets/icon-church.svg";
import homeIcon from "../../assets/icon-home.svg";
import calendarIcon from "../../assets/icon-calendar.svg";
import uploadIcon from "../../assets/icon-upload.svg";
import clipboardIcon from "../../assets/icon-clipboard.svg";
import userCheckIcon from "../../assets/icon-user-check.svg";
import userIcon from "../../assets/icon-user.svg";
import tagIcon from "../../assets/icon-tag.svg";
import fileIcon from "../../assets/icon-file.svg";
import chartIcon from "../../assets/icon-chart.svg";
import historyIcon from "../../assets/icon-history.svg";
import shieldIcon from "../../assets/icon-shield.svg";
import logoutIcon from "../../assets/icon-logout.svg";
import lockIcon from "../../assets/icon-lock.svg";

export default function Sidebar() {
  const navigate = useNavigate();

  const user = getSession()?.user;
  const userRole = user?.role;
  const isMajelis = userRole === "Majelis";

  const [totalJemaat, setTotalJemaat] = useState(null);
  const [counterError, setCounterError] = useState(false);

  useEffect(() => {
    if (userRole !== "Super Admin") {
      setTotalJemaat(null);
      return;
    }

    const controller = new AbortController();
    let running = false;
    let queued = false;

    async function updateCounter() {
      if (controller.signal.aborted || document.hidden) return;

      if (running) {
        queued = true;
        return;
      }

      running = true;

      try {
        const response = await api.get("/users", {
          params: { page: 1, limit: 1 },
          signal: controller.signal,
        });

        if (controller.signal.aborted) return;

        const total = response.data.data?.meta?.total;

        if (!Number.isInteger(total) || total < 0) {
          throw new Error("Total jemaat tidak valid");
        }

        setTotalJemaat(total);
        setCounterError(false);
      } catch {
        if (!controller.signal.aborted) {
          setCounterError(true);
        }
      } finally {
        running = false;

        if (queued && !controller.signal.aborted) {
          queued = false;
          void updateCounter();
        }
      }
    }

    function onVisible() {
      if (!document.hidden) void updateCounter();
    }

    void updateCounter();

    const timer = window.setInterval(updateCounter, 30000);

    window.addEventListener("jemaat-updated", updateCounter);
    window.addEventListener("focus", updateCounter);
    document.addEventListener("visibilitychange", onVisible);

    return () => {
      controller.abort();
      window.clearInterval(timer);
      window.removeEventListener("jemaat-updated", updateCounter);
      window.removeEventListener("focus", updateCounter);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [userRole]);

  const initials = (user?.nama || "")
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((s) => s[0] || "")
    .join("")
    .toUpperCase();

  const handleLogout = async () => {
    const pending = logout();
    navigate("/login", { replace: true });
    await pending;
  };

  const navLinkStyle = ({ isActive }) => ({
    display: "flex",
    alignItems: "center",
    gap: "12px",
    padding: "9px 12px",
    borderRadius: "8px",
    textDecoration: "none",
    fontSize: "13px",
    cursor: "pointer",
    transition: "background 0.15s ease",
    backgroundColor: isActive ? "#17403F" : "transparent",
    color: isActive ? "#E8F1F1" : "#93AFB1",
    fontWeight: isActive ? "600" : "400",
  });

  const disabledStyle = {
    display: "flex",
    alignItems: "center",
    gap: "12px",
    padding: "9px 12px",
    borderRadius: "8px",
    fontSize: "13px",
    color: "#93AFB1",
    opacity: 0.45,
    cursor: "not-allowed",
    userSelect: "none",
  };

  return (
    <aside style={styles.sidebar}>
      <div style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
        {/* Brand */}
        <div style={styles.brandHeader}>
          <div style={styles.logoBadge}>
            <img src={churchIcon} style={styles.iconMd} alt="Logo" />
          </div>
          <div style={{ display: "flex", flexDirection: "column" }}>
            <span style={styles.brandTitle}>GKKA Tenggilis</span>
            <span style={styles.brandSubtitle}>Sistem manajemen gereja</span>
          </div>
        </div>

        {/* Beranda */}
        <NavLink to="/" style={navLinkStyle}>
          <img src={homeIcon} style={styles.iconMd} alt="Beranda" />
          <span style={{ flex: 1 }}>Beranda</span>
        </NavLink>

        {/* Section: Ibadah & Pelayanan */}
        <div style={styles.sectionHeader}>Ibadah & pelayanan</div>
        <NavLink to="/jadwal-kegiatan" style={navLinkStyle}>
          <img src={calendarIcon} style={styles.iconMd} alt="Jadwal" />
          <span style={{ flex: 1 }}>Jadwal kegiatan</span>
        </NavLink>

        {isMajelis ? (
          <>
            <div style={disabledStyle} title="Terkunci untuk Majelis">
              <img src={lockIcon} style={styles.iconMd} alt="Terkunci" />
              <span style={{ flex: 1 }}>Impor Excel</span>
            </div>
            <div style={disabledStyle} title="Terkunci untuk Majelis">
              <img src={lockIcon} style={styles.iconMd} alt="Terkunci" />
              <span style={{ flex: 1 }}>Penugasan pelayan</span>
            </div>
            <div style={disabledStyle} title="Terkunci untuk Majelis">
              <img src={lockIcon} style={styles.iconMd} alt="Terkunci" />
              <span style={{ flex: 1 }}>Absensi</span>
            </div>
          </>
        ) : (
          <>
            <NavLink to="/impor-jadwal" style={navLinkStyle}>
              <img src={uploadIcon} style={styles.iconMd} alt="Impor" />
              <span style={{ flex: 1 }}>Impor Excel</span>
            </NavLink>
            <NavLink to="/penugasan-pelayan" style={navLinkStyle}>
              <img src={clipboardIcon} style={styles.iconMd} alt="Penugasan" />
              <span style={{ flex: 1 }}>Penugasan pelayan</span>
              <span style={styles.badgeDanger}>2</span>
            </NavLink>
            <NavLink to="/absensi" style={navLinkStyle}>
              <img src={userCheckIcon} style={styles.iconMd} alt="Absensi" />
              <span style={{ flex: 1 }}>Absensi</span>
            </NavLink>
          </>
        )}

        {/* Section: Jemaat */}
        <div style={styles.sectionHeader}>Jemaat</div>
        <NavLink to="/data-jemaat" style={navLinkStyle}>
          <img src={userIcon} style={styles.iconMd} alt="" />
          <span style={{ flex: 1 }}>Data jemaat</span>

          {userRole === "Super Admin" && (
            <span
              style={styles.badgeMuted}
              aria-live="polite"
              title={
                counterError
                  ? "Pembaruan gagal. Jumlah terakhir yang berhasil dimuat."
                  : "Total akun pada Data Jemaat"
              }
            >
              {totalJemaat === null
                ? counterError
                  ? "—"
                  : "…"
                : totalJemaat.toLocaleString("id-ID")}
              {counterError && totalJemaat !== null ? " *" : ""}
            </span>
          )}
        </NavLink>

        {userRole === "Super Admin" && (
          <NavLink to="/komunitas-keahlian" style={navLinkStyle}>
            <img src={tagIcon} style={styles.iconMd} alt="" />
            <span style={{ flex: 1 }}>Komunitas & keahlian</span>
          </NavLink>
        )}


        {/* Section: Evaluasi */}
        <div style={styles.sectionHeader}>Evaluasi</div>
        {isMajelis ? (
          <>
            <div style={disabledStyle} title="Terkunci untuk Majelis">
              <img src={lockIcon} style={styles.iconMd} alt="Terkunci" />
              <span style={{ flex: 1 }}>Kuesioner</span>
            </div>
            <NavLink to="/laporan-majelis" style={navLinkStyle}>
              <img src={chartIcon} style={styles.iconMd} alt="Laporan" />
              <span style={{ flex: 1 }}>Laporan evaluasi</span>
            </NavLink>
          </>
        ) : (
          <>
            <NavLink to="/kuesioner" style={navLinkStyle}>
              <img src={fileIcon} style={styles.iconMd} alt="Kuesioner" />
              <span style={{ flex: 1 }}>Kuesioner</span>
            </NavLink>
            <NavLink to="/laporan-evaluasi" style={navLinkStyle}>
              <img src={chartIcon} style={styles.iconMd} alt="Laporan" />
              <span style={{ flex: 1 }}>Laporan evaluasi</span>
            </NavLink>
          </>
        )}

        {/* Section: Sistem */}
        <div style={styles.sectionHeader}>Sistem</div>
        {isMajelis ? (
          <>
            <div style={disabledStyle} title="Terkunci untuk Majelis">
              <img src={lockIcon} style={styles.iconMd} alt="Terkunci" />
              <span style={{ flex: 1 }}>Audit trail</span>
            </div>
            <div style={disabledStyle} title="Terkunci untuk Majelis">
              <img src={lockIcon} style={styles.iconMd} alt="Terkunci" />
              <span style={{ flex: 1 }}>Hak akses</span>
            </div>
          </>
        ) : (
          <>
            <NavLink to="/audit-trail" style={navLinkStyle}>
              <img src={historyIcon} style={styles.iconMd} alt="Audit" />
              <span style={{ flex: 1 }}>Audit trail</span>
            </NavLink>
            <NavLink to="/hak-akses" style={navLinkStyle}>
              <img src={shieldIcon} style={styles.iconMd} alt="Akses" />
              <span style={{ flex: 1 }}>Hak akses</span>
            </NavLink>
          </>
        )}
      </div>

      {/* Profil Pengguna */}
      <div style={styles.userProfile}>
        <div style={styles.avatar}>{initials || "?"}</div>
        <div style={{ flex: 1, display: "flex", flexDirection: "column" }}>
          <span
            style={{
              color: "#E8F1F1",
              fontSize: "13px",
              fontWeight: "600",
              lineHeight: "18px",
            }}
          >
            {user?.nama || "Pengguna"}
          </span>
          <span
            style={{ color: "#93AFB1", fontSize: "11px", fontWeight: "500" }}
          >
            {userRole || ""}
          </span>
        </div>
        <button
          onClick={handleLogout}
          title="Keluar"
          style={styles.logoutButton}
        >
          <img src={logoutIcon} style={styles.iconSm} alt="Logout" />
        </button>
      </div>
    </aside>
  );
}

const styles = {
  sidebar: {
    width: "256px",
    minWidth: "256px",
    height: "100vh",
    maxHeight: "100vh",
    flexShrink: 0 /* Menjaga lebar sidebar tetap saat di-zoom */,
    backgroundColor: "#0D2B2E",
    display: "flex",
    flexDirection: "column",
    justifyContent: "space-between",
    padding: "20px 16px 16px 16px",
    borderRight: "1px solid #17403F",
    overflowY: "auto",
    boxSizing: "border-box",
  },
  brandHeader: {
    display: "flex",
    alignItems: "center",
    gap: "12px",
    padding: "4px 8px 20px 8px",
  },
  logoBadge: {
    width: "36px",
    height: "36px",
    backgroundColor: "#C9952B",
    borderRadius: "8px",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  },
  brandTitle: {
    color: "#E8F1F1",
    fontSize: "14px",
    fontWeight: "600",
    lineHeight: "20px",
  },
  brandSubtitle: {
    color: "#93AFB1",
    fontSize: "12px",
    fontWeight: "500",
    lineHeight: "16px",
  },
  sectionHeader: {
    color: "#93AFB1",
    fontSize: "11px",
    fontWeight: "600",
    textTransform: "uppercase",
    letterSpacing: "0.5px",
    padding: "16px 12px 6px 12px",
  },
  badgeDanger: {
    padding: "1px 6px",
    backgroundColor: "#D6334A",
    borderRadius: "4px",
    color: "white",
    fontSize: "11px",
    fontWeight: "600",
  },
  badgeMuted: {
    padding: "1px 6px",
    backgroundColor: "#17403F",
    borderRadius: "4px",
    color: "#E8F1F1",
    fontSize: "11px",
    fontWeight: "500",
  },
  userProfile: {
    padding: "10px 12px",
    backgroundColor: "#17403F",
    borderRadius: "12px",
    display: "flex",
    alignItems: "center",
    gap: "10px",
    marginTop: "24px",
  },
  avatar: {
    width: "32px",
    height: "32px",
    backgroundColor: "#FBF2DD",
    borderRadius: "50%",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: "12px",
    fontWeight: "700",
    color: "#8A6000",
  },
  logoutButton: {
    background: "none",
    border: "none",
    cursor: "pointer",
    opacity: 0.8,
    display: "flex",
    alignItems: "center",
    padding: 0,
  },
  iconSm: {
    width: "18px",
    height: "18px",
  },
  iconMd: {
    width: "20px",
    height: "20px",
  },
};
