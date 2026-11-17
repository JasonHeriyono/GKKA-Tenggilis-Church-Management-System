import { useEffect, useId, useRef, useState } from "react";
import api from "../../services/api";

import calendarIcon from "../../assets/icon-calendar.svg";
import leftIcon from "../../assets/icon-chevron-left.svg";
import rightIcon from "../../assets/icon-chevron-right.svg";
import downIcon from "../../assets/icon-chevron-down.svg";

const dayNames = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"];

function dateKey(date) {
  const pad = (value) => String(value).padStart(2, "0");

  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function parseDate(value) {
  return new Date(`${value}T12:00:00`);
}

export default function KalenderKegiatan({
  value,
  onChange,
  disabled = false,
}) {
  const [open, setOpen] = useState(false);
  const [month, setMonth] = useState(() => {
    const date = parseDate(value);
    return new Date(date.getFullYear(), date.getMonth(), 1, 12);
  });

  const [counts, setCounts] = useState({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [revision, setRevision] = useState(0);

  const rootRef = useRef(null);
  const triggerRef = useRef(null);
  const popupRef = useRef(null);
  const popupID = useId();

  const monthStart = dateKey(month);
  const monthEnd = dateKey(
    new Date(month.getFullYear(), month.getMonth() + 1, 0, 12),
  );

  const firstWeekday = (month.getDay() + 6) % 7;
  const totalDays = new Date(
    month.getFullYear(),
    month.getMonth() + 1,
    0,
  ).getDate();

  const todayKey = dateKey(new Date());
  const selectedLabel = parseDate(value).toLocaleDateString("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });

  const monthLabel = month.toLocaleDateString("id-ID", {
    month: "long",
    year: "numeric",
  });

  useEffect(() => {
    if (!open) return;

    const controller = new AbortController();

    async function load() {
      setLoading(true);
      setError("");
      setCounts({});

      try {
        const nextCounts = {};
        let page = 1;

        while (true) {
          const response = await api.get("/kegiatan", {
            params: {
              dari: monthStart,
              sampai: monthEnd,
              page,
              limit: 100,
            },
            signal: controller.signal,
          });

          if (controller.signal.aborted) return;

          const data = response.data.data;

          for (const activity of data.items || []) {
            const key = activity.tanggal;
            nextCounts[key] = (nextCounts[key] || 0) + 1;
          }

          if (page >= (data.meta?.total_pages || 1)) break;
          page += 1;
        }

        setCounts(nextCounts);
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(
            err.response?.data?.message ||
              "Jumlah kegiatan belum dapat dimuat.",
          );
        }
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }

    load();
    return () => controller.abort();
  }, [open, monthStart, monthEnd, revision]);

  useEffect(() => {
    if (!open) return;

    popupRef.current?.focus();

    function outsideClick(event) {
      if (!rootRef.current?.contains(event.target)) setOpen(false);
    }

    function escape(event) {
      if (event.key === "Escape") {
        setOpen(false);
        triggerRef.current?.focus();
      }
    }

    function focusOutside(event) {
      if (!rootRef.current?.contains(event.target)) setOpen(false);
    }

    document.addEventListener("pointerdown", outsideClick);
    document.addEventListener("keydown", escape);
    document.addEventListener("focusin", focusOutside);

    return () => {
      document.removeEventListener("pointerdown", outsideClick);
      document.removeEventListener("keydown", escape);
      document.removeEventListener("focusin", focusOutside);
    };
  }, [open]);

  useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled]);

  function toggleCalendar() {
    if (disabled) return;

    if (!open) {
      const date = parseDate(value);
      setMonth(new Date(date.getFullYear(), date.getMonth(), 1, 12));
    }

    setOpen((previous) => !previous);
  }

  function moveMonth(offset) {
    setMonth(
      (previous) =>
        new Date(previous.getFullYear(), previous.getMonth() + offset, 1, 12),
    );
  }

  function selectDate(key) {
    onChange(key);
    setOpen(false);
    triggerRef.current?.focus();
  }

  return (
    <div ref={rootRef} style={styles.wrapper}>
      <button
        ref={triggerRef}
        type="button"
        disabled={disabled}
        onClick={toggleCalendar}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-controls={popupID}
        style={{
          ...styles.trigger,
          borderColor: open ? "#00777C" : "#CBD6D6",
          boxShadow: open ? "0 0 0 3px #E0F3F2" : "none",
          opacity: disabled ? 0.55 : 1,
        }}
      >
        <span style={styles.iconBadge}>
          <img src={calendarIcon} alt="" style={styles.icon} />
        </span>

        <span style={styles.triggerText}>
          <span style={styles.caption}>Tanggal kegiatan</span>
          <strong>{selectedLabel}</strong>
        </span>

        <img
          src={downIcon}
          alt=""
          style={{
            ...styles.smallIcon,
            transform: open ? "rotate(180deg)" : "none",
          }}
        />
      </button>

      {open && (
        <div
          id={popupID}
          ref={popupRef}
          role="dialog"
          aria-label="Pilih tanggal kegiatan"
          tabIndex={-1}
          style={styles.popup}
        >
          <div style={styles.header}>
            <button
              type="button"
              aria-label="Bulan sebelumnya"
              onClick={() => moveMonth(-1)}
              style={styles.arrow}
            >
              <img src={leftIcon} alt="" style={styles.smallIcon} />
            </button>

            <strong style={styles.month}>{monthLabel}</strong>

            <button
              type="button"
              aria-label="Bulan berikutnya"
              onClick={() => moveMonth(1)}
              style={styles.arrow}
            >
              <img src={rightIcon} alt="" style={styles.smallIcon} />
            </button>
          </div>

          <div style={styles.grid}>
            {dayNames.map((day) => (
              <div key={day} style={styles.weekday}>
                {day}
              </div>
            ))}

            {Array.from({ length: firstWeekday }, (_, index) => (
              <div key={`empty-${index}`} aria-hidden="true" />
            ))}

            {Array.from({ length: totalDays }, (_, index) => {
              const day = index + 1;
              const key = dateKey(
                new Date(month.getFullYear(), month.getMonth(), day, 12),
              );

              const count = counts[key] || 0;
              const selected = key === value;
              const isToday = key === todayKey;

              return (
                <button
                  key={key}
                  type="button"
                  onClick={() => selectDate(key)}
                  aria-pressed={selected}
                  aria-current={isToday ? "date" : undefined}
                  aria-label={`${day} ${monthLabel}${
                    loading || error
                      ? ", jumlah kegiatan belum tersedia"
                      : `, ${count} kegiatan`
                  }${isToday ? ", hari ini" : ""}`}
                  title={
                    loading || error
                      ? "Jumlah kegiatan belum tersedia"
                      : `${count} kegiatan`
                  }
                  style={{
                    ...styles.day,
                    background: selected
                      ? "#00777C"
                      : count > 0
                        ? "#E6F5ED"
                        : "#fff",
                    color: selected
                      ? "#fff"
                      : count > 0
                        ? "#166534"
                        : "#334155",
                    borderColor: selected
                      ? "#00777C"
                      : isToday
                        ? "#00777C"
                        : count > 0
                          ? "#B9DFC8"
                          : "transparent",
                  }}
                >
                  <span>{day}</span>

                  {count > 0 && (
                    <span
                      style={{
                        ...styles.counter,
                        background: selected ? "#fff" : "#15803D",
                        color: selected ? "#00777C" : "#fff",
                      }}
                    >
                      {count}
                    </span>
                  )}

                  {isToday && (
                    <span
                      style={{
                        ...styles.todayDot,
                        background: selected ? "#fff" : "#00777C",
                      }}
                    />
                  )}
                </button>
              );
            })}
          </div>

          <div role="status" style={styles.status}>
            {loading
              ? "Memuat jumlah kegiatan…"
              : error ||
                "Tanggal hijau memiliki kegiatan. Angka menunjukkan jumlahnya."}
          </div>

          {error && (
            <button
              type="button"
              style={styles.retry}
              onClick={() => setRevision((previous) => previous + 1)}
            >
              Coba muat ulang
            </button>
          )}

          <div style={styles.footer}>
            <button
              type="button"
              style={styles.todayButton}
              onClick={() => selectDate(todayKey)}
            >
              Hari ini
            </button>

            <button
              type="button"
              style={styles.closeButton}
              onClick={() => {
                setOpen(false);
                triggerRef.current?.focus();
              }}
            >
              Tutup
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

const styles = {
  wrapper: {
    position: "relative",
    maxWidth: "100%",
  },
  trigger: {
    display: "flex",
    alignItems: "center",
    gap: 12,
    padding: "8px 12px",
    border: "1px solid #CBD6D6",
    borderRadius: 12,
    background: "#fff",
    color: "#0D1F21",
    cursor: "pointer",
    fontFamily: "inherit",
  },
  iconBadge: {
    width: 36,
    height: 36,
    borderRadius: 10,
    background: "#E0F3F2",
    display: "grid",
    placeItems: "center",
  },
  icon: { width: 20, height: 20 },
  smallIcon: { width: 16, height: 16 },
  triggerText: {
    display: "flex",
    flexDirection: "column",
    textAlign: "left",
    gap: 3,
    fontSize: 13,
  },
  caption: { fontSize: 10, color: "#66787A" },
  popup: {
    position: "absolute",
    top: "calc(100% + 10px)",
    right: 0,
    width: 336,
    maxWidth: "calc(100vw - 40px)",
    boxSizing: "border-box",
    padding: 16,
    background: "#fff",
    border: "1px solid #E1E8E8",
    borderRadius: 16,
    boxShadow: "0 16px 48px rgba(13,43,46,.18)",
    zIndex: 100,
    outline: "none",
  },
  header: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: 16,
  },
  month: {
    color: "#0D1F21",
    fontSize: 15,
    textTransform: "capitalize",
  },
  arrow: {
    width: 32,
    height: 32,
    border: "1px solid #E1E8E8",
    borderRadius: 8,
    background: "#fff",
    cursor: "pointer",
    display: "grid",
    placeItems: "center",
  },
  grid: {
    display: "grid",
    gridTemplateColumns: "repeat(7, minmax(0, 1fr))",
    gap: 5,
  },
  weekday: {
    textAlign: "center",
    padding: "6px 0",
    color: "#829194",
    fontSize: 11,
    fontWeight: 600,
  },
  day: {
    position: "relative",
    minHeight: 43,
    border: "1px solid transparent",
    borderRadius: 10,
    fontSize: 13,
    fontWeight: 600,
    fontFamily: "inherit",
    cursor: "pointer",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    gap: 2,
    padding: "4px 0",
  },
  counter: {
    minWidth: 14,
    padding: "0 3px",
    borderRadius: 5,
    fontSize: 9,
    lineHeight: "13px",
  },
  todayDot: {
    position: "absolute",
    bottom: 2,
    right: 3,
    width: 4,
    height: 4,
    borderRadius: "50%",
  },
  status: {
    marginTop: 14,
    color: "#66787A",
    fontSize: 11,
    lineHeight: 1.6,
  },
  retry: {
    marginTop: 8,
    padding: 0,
    border: 0,
    background: "none",
    color: "#00777C",
    cursor: "pointer",
    fontFamily: "inherit",
  },
  footer: {
    display: "flex",
    justifyContent: "space-between",
    marginTop: 12,
    paddingTop: 12,
    borderTop: "1px solid #E1E8E8",
  },
  todayButton: {
    padding: "7px 12px",
    background: "#E0F3F2",
    color: "#075E62",
    border: 0,
    borderRadius: 8,
    fontWeight: 600,
    cursor: "pointer",
    fontFamily: "inherit",
  },
  closeButton: {
    padding: "7px 12px",
    background: "#fff",
    color: "#66787A",
    border: "1px solid #E1E8E8",
    borderRadius: 8,
    cursor: "pointer",
    fontFamily: "inherit",
  },
};
