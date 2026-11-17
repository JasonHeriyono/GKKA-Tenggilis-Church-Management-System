import usePenugasanPelayan from "./usePenugasanPelayan";
import KalenderKegiatan from "./KalenderKegiatan";
import refreshIcon from "../../assets/icon-refresh.svg";

export default function PenugasanPelayan() {
  const {
    date,
    changeDate,
    activities,
    activityID,
    activity,
    chooseActivity,
    skills,
    skillID,
    skill,
    chooseSkill,
    assignments,
    visibleCandidates,
    available,
    assignedIDs,
    selectedIDs,
    setSelectedIDs,
    toggleUser,
    toggleAll,
    allSelected,
    search,
    setSearch,
    loadingActivities,
    loadingSkills,
    loadingAssignments,
    loadingCandidates,
    error,
    candidateError,
    notice,
    warnings,
    busy,
    progress,
    assignSelected,
    removeAssignment,
    reload,
  } = usePenugasanPelayan();

  return (
    <div style={styles.page}>
      <div style={styles.header}>
        <div>
          <h1 style={styles.title}>Penugasan pelayan</h1>
          <p style={styles.subtitle}>
            Pilih kegiatan dan keahlian, lalu tugaskan beberapa jemaat
            sekaligus.
          </p>
        </div>

        <div style={styles.row}>
          <KalenderKegiatan
            value={date}
            onChange={changeDate}
            disabled={busy}
          />

          <button
            type="button"
            style={{
              ...styles.secondary,
              display: "flex",
              alignItems: "center",
              gap: 8,
            }}
            disabled={busy || loadingActivities || loadingSkills}
            onClick={reload}
          >
            <img src={refreshIcon} alt="" style={{ width: 16, height: 16 }} />
            Muat ulang
          </button>
        </div>
      </div>

      {error && (
        <div role="alert" style={styles.error}>
          {error}
        </div>
      )}
      {notice && (
        <div role="status" style={styles.success}>
          {notice}
        </div>
      )}

      {warnings.length > 0 && (
        <div role="status" style={styles.warning}>
          <strong>Penugasan tersimpan dengan peringatan bentrok:</strong>
          <ul style={{ marginBottom: 0, paddingLeft: 20 }}>
            {warnings.map((warning, index) => (
              <li key={`${index}-${warning}`}>{warning}</li>
            ))}
          </ul>
        </div>
      )}

      <section style={styles.card}>
        <label style={styles.label}>
          Kegiatan pada tanggal terpilih
          <select
            value={activityID}
            onChange={(event) => chooseActivity(event.target.value)}
            disabled={busy || loadingActivities || activities.length === 0}
            style={styles.input}
          >
            <option value="">
              {loadingActivities
                ? "Memuat kegiatan…"
                : activities.length === 0
                  ? "Tidak ada kegiatan pada tanggal ini"
                  : "Pilih kegiatan"}
            </option>

            {activities.map((item) => (
              <option key={item.id} value={String(item.id)}>
                {item.waktu_mulai} · {item.nama_kegiatan}
                {item.nama_komunitas ? ` · ${item.nama_komunitas}` : ""}
              </option>
            ))}
          </select>
        </label>

        {activity && (
          <div style={styles.subtitle}>
            {activity.waktu_mulai}
            {activity.waktu_selesai ? `–${activity.waktu_selesai}` : ""}
            {" · "}
            {activity.lokasi || "Lokasi belum diisi"}
            {" · "}
            {activity.nama_komunitas || "Kegiatan umum"}
          </div>
        )}
      </section>

      {activityID && (
        <div style={styles.columns}>
          <section style={{ ...styles.card, flex: "1 1 420px", minWidth: 0 }}>
            <div style={styles.header}>
              <h2 style={styles.sectionTitle}>Petugas kegiatan</h2>
              <span style={styles.small}>{assignments.length} penugasan</span>
            </div>

            {loadingAssignments ? (
              <div style={styles.empty}>Memuat penugasan…</div>
            ) : assignments.length === 0 ? (
              <div style={styles.empty}>
                Belum ada petugas. Pilih keahlian di panel penugasan.
              </div>
            ) : (
              <div style={styles.list}>
                {assignments.map((item) => (
                  <div key={item.id_penjadwalan} style={styles.assignment}>
                    <div>
                      <strong>{item.nama}</strong>
                      <div style={styles.small}>
                        {item.nama_keahlian}
                        {" · "}
                        {item.kode_jemaat || "Tanpa kode jemaat"}
                      </div>
                      <div style={styles.small}>{item.status_konfirmasi}</div>
                    </div>

                    <button
                      type="button"
                      style={styles.danger}
                      disabled={busy}
                      onClick={() => removeAssignment(item)}
                    >
                      Batalkan
                    </button>
                  </div>
                ))}
              </div>
            )}
          </section>

          <section style={{ ...styles.card, flex: "1 1 360px", minWidth: 0 }}>
            <h2 style={styles.sectionTitle}>Tambah petugas</h2>

            <label style={styles.label}>
              Keahlian pelayanan
              <select
                value={skillID}
                onChange={(event) => chooseSkill(event.target.value)}
                disabled={busy || loadingSkills || skills.length === 0}
                style={styles.input}
              >
                <option value="">
                  {loadingSkills
                    ? "Memuat keahlian…"
                    : skills.length === 0
                      ? "Belum ada keahlian"
                      : "Pilih keahlian"}
                </option>

                {skills.map((item) => (
                  <option key={item.id} value={String(item.id)}>
                    {item.nama}
                  </option>
                ))}
              </select>
            </label>

            {skillID && (
              <>
                <input
                  type="search"
                  aria-label="Cari calon petugas"
                  placeholder={`Cari nama atau kode pemilik keahlian ${skill?.nama || ""}`}
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  disabled={busy}
                  style={styles.input}
                />

                <div style={styles.header}>
                  <label style={styles.row}>
                    <input
                      type="checkbox"
                      checked={allSelected}
                      disabled={
                        busy ||
                        loadingCandidates ||
                        loadingAssignments ||
                        available.length === 0
                      }
                      onChange={toggleAll}
                    />
                    Pilih semua hasil
                  </label>

                  <span style={styles.small}>
                    {selectedIDs.length} terpilih
                  </span>
                </div>

                {selectedIDs.length > 0 && (
                  <button
                    type="button"
                    style={{ ...styles.secondary, alignSelf: "flex-start" }}
                    disabled={busy}
                    onClick={() => setSelectedIDs([])}
                  >
                    Bersihkan pilihan
                  </button>
                )}

                {candidateError && (
                  <div role="alert" style={styles.error}>
                    {candidateError}
                  </div>
                )}

                <div style={styles.candidates}>
                  {loadingCandidates || loadingAssignments ? (
                    <div style={styles.empty}>Memuat calon petugas…</div>
                  ) : visibleCandidates.length === 0 ? (
                    <div style={styles.empty}>
                      Tidak ada jemaat sesuai keahlian dan pencarian.
                    </div>
                  ) : (
                    visibleCandidates.map((user) => {
                      const assigned = assignedIDs.has(user.id_user);

                      return (
                        <label
                          key={user.id_user}
                          style={{
                            ...styles.candidate,
                            background: selectedIDs.includes(user.id_user)
                              ? "#F0F9F9"
                              : "#fff",
                            opacity: assigned ? 0.6 : 1,
                          }}
                        >
                          <input
                            type="checkbox"
                            checked={selectedIDs.includes(user.id_user)}
                            disabled={busy || assigned}
                            onChange={() => toggleUser(user.id_user)}
                          />

                          <span>
                            <strong>{user.nama}</strong>
                            <span style={{ ...styles.small, display: "block" }}>
                              {user.kode_jemaat || "Tanpa kode jemaat"}
                            </span>
                            {assigned && (
                              <span
                                style={{ ...styles.small, display: "block" }}
                              >
                                Sudah ditugaskan pada keahlian ini
                              </span>
                            )}
                          </span>
                        </label>
                      );
                    })
                  )}
                </div>

                <button
                  type="button"
                  style={styles.primary}
                  disabled={
                    busy ||
                    selectedIDs.length === 0 ||
                    loadingAssignments ||
                    loadingCandidates ||
                    Boolean(candidateError)
                  }
                  onClick={assignSelected}
                >
                  {busy && progress
                    ? `Memproses ${progress.completed}/${progress.total}…`
                    : `Tugaskan ${selectedIDs.length} jemaat`}
                </button>
              </>
            )}
          </section>
        </div>
      )}
    </div>
  );
}

const button = {
  padding: "9px 14px",
  borderRadius: 8,
  fontSize: 13,
  fontWeight: 600,
  fontFamily: "inherit",
  cursor: "pointer",
};

const styles = {
  page: {
    display: "flex",
    flexDirection: "column",
    gap: 20,
  },
  header: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    flexWrap: "wrap",
    gap: 12,
  },
  title: {
    margin: 0,
    fontSize: 28,
    color: "#0D1F21",
  },
  subtitle: {
    margin: "6px 0 0",
    fontSize: 13,
    color: "#66787A",
    lineHeight: 1.6,
  },
  row: {
    display: "flex",
    alignItems: "center",
    flexWrap: "wrap",
    gap: 8,
    fontSize: 13,
  },
  columns: {
    display: "flex",
    flexWrap: "wrap",
    alignItems: "flex-start",
    gap: 20,
  },
  card: {
    padding: 20,
    border: "1px solid #E1E8E8",
    borderRadius: 12,
    background: "#fff",
    display: "flex",
    flexDirection: "column",
    gap: 16,
  },
  sectionTitle: {
    margin: 0,
    fontSize: 16,
    color: "#0D1F21",
  },
  label: {
    display: "flex",
    flexDirection: "column",
    gap: 8,
    color: "#445759",
    fontSize: 13,
    fontWeight: 600,
  },
  input: {
    padding: "10px 12px",
    border: "1px solid #CBD6D6",
    borderRadius: 8,
    background: "#fff",
    color: "#0D1F21",
    fontSize: 14,
    fontFamily: "inherit",
    minWidth: 0,
    boxSizing: "border-box",
  },
  primary: {
    ...button,
    border: "1px solid #00777C",
    background: "#00777C",
    color: "#fff",
  },
  secondary: {
    ...button,
    border: "1px solid #CBD6D6",
    background: "#fff",
    color: "#0D1F21",
  },
  danger: {
    ...button,
    border: "1px solid #F3B9B9",
    background: "#FFF3F3",
    color: "#B42318",
  },
  small: {
    marginTop: 4,
    fontSize: 12,
    color: "#66787A",
    lineHeight: 1.5,
  },
  list: {
    display: "flex",
    flexDirection: "column",
    maxHeight: 560,
    overflowY: "auto",
  },
  assignment: {
    padding: "14px 0",
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    gap: 12,
    borderBottom: "1px solid #E1E8E8",
    fontSize: 14,
  },
  candidates: {
    display: "flex",
    flexDirection: "column",
    gap: 8,
    maxHeight: 360,
    overflowY: "auto",
  },
  candidate: {
    padding: 12,
    display: "flex",
    alignItems: "flex-start",
    gap: 10,
    border: "1px solid #E1E8E8",
    borderRadius: 8,
    fontSize: 14,
    color: "#0D1F21",
  },
  empty: {
    padding: "40px 16px",
    textAlign: "center",
    color: "#66787A",
    fontSize: 13,
  },
  error: {
    padding: 14,
    borderRadius: 8,
    color: "#B42318",
    background: "#FFF0F0",
    whiteSpace: "pre-wrap",
    fontSize: 13,
  },
  success: {
    padding: 14,
    borderRadius: 8,
    color: "#09703E",
    background: "#E0F3E8",
    fontSize: 13,
  },
  warning: {
    padding: 16,
    borderRadius: 8,
    color: "#92400E",
    background: "#FFF4DE",
    fontSize: 13,
    lineHeight: 1.6,
  },
};
