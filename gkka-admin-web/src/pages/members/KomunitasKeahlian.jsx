import { useEffect, useRef, useState } from "react";
import apiClient from "../../services/api";

const initialForm = { nama: "", deskripsi: "" };

function errorText(error) {
  const body = error.response?.data;
  const fields = Object.entries(body?.errors || {})
    .map(([key, value]) => `${key}: ${value}`)
    .join("\n");

  return [body?.message || "Permintaan gagal. Coba kembali.", fields]
    .filter(Boolean)
    .join("\n");
}

export default function KomunitasKeahlian() {
  const [type, setType] = useState("komunitas");
  const [items, setItems] = useState([]);
  const [meta, setMeta] = useState(null);
  const [page, setPage] = useState(1);
  const [searchInput, setSearchInput] = useState("");
  const [query, setQuery] = useState("");
  const [revision, setRevision] = useState(0);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const [detail, setDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const detailVersion = useRef(0);

  // null | create | edit | member
  const [modal, setModal] = useState(null);
  const [form, setForm] = useState(initialForm);
  const [modalError, setModalError] = useState("");
  const [busy, setBusy] = useState(false);
  const mutationLock = useRef(false);

  const [userInput, setUserInput] = useState("");
  const [userQuery, setUserQuery] = useState("");
  const [userPage, setUserPage] = useState(1);
  const [users, setUsers] = useState([]);
  const [userMeta, setUserMeta] = useState(null);
  const [usersLoading, setUsersLoading] = useState(false);
  const [usersError, setUsersError] = useState("");
  const [selectedUserIDs, setSelectedUserIDs] = useState([]);
  const [isKetua, setIsKetua] = useState(false);
  const [bulkProgress, setBulkProgress] = useState(null);

  function toggleSelectedUser(userID) {
    setSelectedUserIDs((previous) =>
      previous.includes(userID)
        ? previous.filter((id) => id !== userID)
        : [...previous, userID],
    );
  }

  const isGroup = type === "komunitas";
  const label = isGroup ? "Komunitas" : "Keahlian";
  const endpoint = `/${type}`;

  const members = detail
    ? (isGroup ? detail.anggota : detail.jemaat) || []
    : [];

  const availableUsers = users.filter(
    (user) => !members.some((member) => member.id_user === user.id),
  );

  const allVisibleSelected =
    availableUsers.length > 0 &&
    availableUsers.every((user) => selectedUserIDs.includes(user.id));

  useEffect(() => {
    const controller = new AbortController();

    async function loadList() {
      setLoading(true);
      setError("");

      try {
        const response = await apiClient.get(`/${type}`, {
          params: { page, limit: 20, q: query },
          signal: controller.signal,
        });

        if (controller.signal.aborted) return;
        setItems(response.data.data.items || []);
        setMeta(response.data.data.meta);
      } catch (err) {
        if (controller.signal.aborted) return;

        if (err.response?.status === 404 && page > 1) {
          setPage((value) => Math.max(1, value - 1));
          return;
        }

        setItems([]);
        setMeta(null);
        setError(errorText(err));
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }

    loadList();
    return () => controller.abort();
  }, [type, page, query, revision]);

  useEffect(() => {
    if (modal !== "member") return;

    const controller = new AbortController();

    async function loadUsers() {
      setUsersLoading(true);
      setUsersError("");
      setUsers([]);
      setUserMeta(null);

      try {
        const response = await apiClient.get("/users", {
          params: { page: userPage, limit: 20, q: userQuery },
          signal: controller.signal,
        });

        if (controller.signal.aborted) return;
        setUsers(response.data.data.items || []);
        setUserMeta(response.data.data.meta);
      } catch (err) {
        if (!controller.signal.aborted) setUsersError(errorText(err));
      } finally {
        if (!controller.signal.aborted) setUsersLoading(false);
      }
    }

    loadUsers();
    return () => controller.abort();
  }, [modal, userPage, userQuery]);

  function refreshList() {
    setRevision((value) => value + 1);
  }

  function closeDetail() {
    detailVersion.current += 1;
    setDetail(null);
    setDetailLoading(false);
  }

  function changeType(next) {
    if (busy || next === type) return;

    closeDetail();
    setModal(null);
    setItems([]);
    setMeta(null);
    setLoading(true);
    setType(next);
    setPage(1);
    setSearchInput("");
    setQuery("");
    setError("");
    setNotice("");
  }

  async function openDetail(item) {
    const version = ++detailVersion.current;
    setDetail(null);
    setDetailLoading(true);
    setError("");

    try {
      const response = await apiClient.get(`${endpoint}/${item.id}`);
      if (version === detailVersion.current) {
        setDetail(response.data.data);
      }
    } catch (err) {
      if (version === detailVersion.current) setError(errorText(err));
    } finally {
      if (version === detailVersion.current) setDetailLoading(false);
    }
  }

  function openForm(mode) {
    setModalError("");
    setNotice("");
    setForm(
      mode === "edit"
        ? { nama: detail.nama, deskripsi: detail.deskripsi || "" }
        : { ...initialForm },
    );
    setModal(mode);
  }

  function toggleVisibleUsers() {
    const visibleIDs = users
      .filter((user) => !members.some((member) => member.id_user === user.id))
      .map((user) => user.id);

    setSelectedUserIDs((previous) => {
      const allSelected =
        visibleIDs.length > 0 &&
        visibleIDs.every((id) => previous.includes(id));

      if (allSelected) {
        return previous.filter((id) => !visibleIDs.includes(id));
      }

      return [...new Set([...previous, ...visibleIDs])];
    });
  }

  function openMemberModal() {
    setModalError("");
    setUsersError("");
    setUserInput("");
    setUserQuery("");
    setUserPage(1);
    setUsers([]);
    setUserMeta(null);
    setUsersLoading(true);
    setSelectedUserIDs([]);
    setBulkProgress(null);
    setIsKetua(false);
    setModal("member");
  }

  async function saveForm(event) {
    event.preventDefault();
    if (mutationLock.current) return;

    if (!form.nama.trim()) {
      setModalError("Nama wajib diisi.");
      return;
    }

    const editing = modal === "edit";
    const previousDetail = detail;

    mutationLock.current = true;
    setBusy(true);
    setModalError("");
    setNotice("");

    try {
      const payload = {
        nama: form.nama.trim(),
        ...(isGroup ? { deskripsi: form.deskripsi.trim() } : {}),
      };

      const response = editing
        ? await apiClient.put(`${endpoint}/${previousDetail.id}`, payload)
        : await apiClient.post(endpoint, payload);

      const saved = response.data.data;

      detailVersion.current += 1;
      setDetailLoading(false);

      setDetail({
        ...(editing ? previousDetail : {}),
        ...saved,
        ...(isGroup
          ? { anggota: editing ? previousDetail.anggota || [] : [] }
          : { jemaat: editing ? previousDetail.jemaat || [] : [] }),
      });

      setModal(null);
      setNotice(`${label} "${saved.nama}" berhasil disimpan.`);
      refreshList();
    } catch (err) {
      setModalError(errorText(err));
    } finally {
      mutationLock.current = false;
      setBusy(false);
    }
  }

  async function deleteItem() {
    if (!detail || mutationLock.current) return;
    if (!window.confirm(`Hapus ${label.toLowerCase()} "${detail.nama}"?`)) {
      return;
    }

    mutationLock.current = true;
    setBusy(true);
    setError("");
    setNotice("");

    try {
      await apiClient.delete(`${endpoint}/${detail.id}`);
      closeDetail();
      setNotice(`${label} berhasil dihapus.`);
      refreshList();
    } catch (err) {
      setError(errorText(err));
    } finally {
      mutationLock.current = false;
      setBusy(false);
    }
  }

  // Jika perubahan tersimpan tetapi refresh detail gagal, jangan ulangi mutasi.
  async function reloadMembers(targetID) {
    try {
      const response = await apiClient.get(`${endpoint}/${targetID}`);
      setDetail(response.data.data);
    } catch (err) {
      closeDetail();
      setError(
        `Perubahan sudah tersimpan, tetapi detail gagal dimuat ulang.\n${errorText(err)}`,
      );
    }
  }

  async function addMember(event) {
    event.preventDefault();

    if (!detail || selectedUserIDs.length === 0 || mutationLock.current) {
      return;
    }

    const targetID = detail.id;
    const selectedIDs = [...selectedUserIDs];
    const suffix = isGroup ? "anggota" : "jemaat";

    const getMembers = (data) => (isGroup ? data.anggota : data.jemaat) || [];

    const userNames = new Map(users.map((user) => [user.id, user.nama]));

    mutationLock.current = true;
    setBusy(true);
    setModalError("");
    setError("");
    setNotice("");
    setBulkProgress({ completed: 0, total: selectedIDs.length });

    let addedCount = 0;
    let existingCount = 0;
    let completed = 0;
    let latestDetail = null;
    let stopped = false;

    const resolvedIDs = new Set();
    const failures = [];

    function markResolved(userID) {
      resolvedIDs.add(userID);

      setSelectedUserIDs((previous) => previous.filter((id) => id !== userID));
    }

    async function fetchLatestDetail() {
      const response = await apiClient.get(`${endpoint}/${targetID}`);
      latestDetail = response.data.data;
      setDetail(latestDetail);
      return latestDetail;
    }

    try {
      // Cek ulang keanggotaan sebelum batch dimulai.
      await fetchLatestDetail();

      for (const userID of selectedIDs) {
        const alreadyExists = getMembers(latestDetail).some(
          (member) => member.id_user === userID,
        );

        if (alreadyExists) {
          existingCount += 1;
          markResolved(userID);
          completed += 1;

          setBulkProgress({
            completed,
            total: selectedIDs.length,
          });

          continue;
        }

        try {
          const response = await apiClient.post(
            `${endpoint}/${targetID}/${suffix}`,
            {
              id_user: userID,
              ...(isGroup ? { is_ketua: isKetua } : {}),
            },
          );

          latestDetail = response.data.data;
          setDetail(latestDetail);
          addedCount += 1;
          markResolved(userID);
        } catch (err) {
          const status = err.response?.status;
          let verifiedExists = false;

          // Request bisa tersimpan di server meskipun respons gagal diterima.
          // Untuk konflik atau hasil yang tidak pasti, cek sebelum mencoba ulang.
          if (!err.response || status === 409 || status >= 500) {
            try {
              const current = await fetchLatestDetail();

              verifiedExists = getMembers(current).some(
                (member) => member.id_user === userID,
              );
            } catch {
              // Pilihan tetap dipertahankan jika hasil belum bisa dipastikan.
            }
          }

          if (verifiedExists) {
            existingCount += 1;
            markResolved(userID);
          } else {
            failures.push({
              userID,
              name: userNames.get(userID) || `Jemaat ID ${userID}`,
              message: errorText(err),
            });
          }

          if (
            !err.response ||
            status === 401 ||
            status === 403 ||
            status === 404 ||
            status === 429 ||
            status >= 500
          ) {
            stopped = true;
          }
        }

        completed += 1;

        setBulkProgress({
          completed,
          total: selectedIDs.length,
        });

        if (stopped) break;
      }

      const remainingIDs = selectedIDs.filter(
        (userID) => !resolvedIDs.has(userID),
      );

      if (remainingIDs.length === 0) {
        setModal(null);

        setNotice(
          [
            isGroup
              ? `${addedCount} anggota baru berhasil ditambahkan.`
              : `Keahlian berhasil diberikan kepada ${addedCount} jemaat baru.`,
            existingCount > 0
              ? `${existingCount} jemaat lainnya sudah terdaftar.`
              : "",
          ]
            .filter(Boolean)
            .join(" "),
        );
      } else {
        setModalError(
          [
            `${resolvedIDs.size} dari ${selectedIDs.length} pilihan sudah terdaftar.`,
            `${remainingIDs.length} pilihan belum selesai dan tetap dipilih.`,
            stopped
              ? "Proses dihentikan. Periksa koneksi atau pesan di bawah sebelum mencoba kembali."
              : "Perbaiki kendala berikut, lalu simpan kembali.",
            ...failures.map((failure) => `${failure.name}: ${failure.message}`),
          ]
            .filter(Boolean)
            .join("\n"),
        );
      }
    } catch (err) {
      setModalError(
        `Belum dapat memeriksa anggota terbaru. Pilihan tetap tersimpan.\n${errorText(err)}`,
      );
    } finally {
      mutationLock.current = false;
      setBusy(false);
      setBulkProgress(null);
    }
  }

  async function toggleKetua(member) {
    if (!detail || mutationLock.current) return;

    const targetID = detail.id;
    mutationLock.current = true;
    setBusy(true);
    setError("");
    setNotice("");

    try {
      const response = await apiClient.patch(
        `/komunitas/${targetID}/anggota/${member.id_user}`,
        { is_ketua: !member.is_ketua },
      );

      setDetail(response.data.data);
      setNotice("Status ketua berhasil diperbarui.");
    } catch (err) {
      setError(errorText(err));
    } finally {
      mutationLock.current = false;
      setBusy(false);
    }
  }

  async function removeMember(member) {
    if (!detail || mutationLock.current) return;

    const question = isGroup
      ? `Keluarkan "${member.nama}" dari komunitas ini?`
      : `Lepas keahlian "${detail.nama}" dari "${member.nama}"?`;

    if (!window.confirm(question)) return;

    const targetID = detail.id;
    mutationLock.current = true;
    setBusy(true);
    setError("");
    setNotice("");

    try {
      const suffix = isGroup ? "anggota" : "jemaat";
      await apiClient.delete(
        `${endpoint}/${targetID}/${suffix}/${member.id_user}`,
      );

      setNotice(
        isGroup
          ? "Anggota berhasil dikeluarkan."
          : "Keahlian berhasil dilepas.",
      );
      await reloadMembers(targetID);
    } catch (err) {
      setError(errorText(err));
    } finally {
      mutationLock.current = false;
      setBusy(false);
    }
  }

  function searchList(event) {
    event.preventDefault();
    closeDetail();
    setPage(1);
    setQuery(searchInput.trim());
    refreshList();
  }

  useEffect(() => {
    if (modal !== "member" || busy) return;

    const nextQuery = userInput.trim();
    if (nextQuery === userQuery) return;

    const timer = setTimeout(() => {
      setUserPage(1);
      setUserQuery(nextQuery);
    }, 350);

    return () => clearTimeout(timer);
  }, [modal, busy, userInput, userQuery]);

  function searchUsers(event) {
    event.preventDefault();
    setUserPage(1);
    setUserQuery(userInput.trim());
  }

  const title =
    modal === "member"
      ? isGroup
        ? "Tambah anggota"
        : "Beri keahlian ke jemaat"
      : `${modal === "edit" ? "Edit" : "Tambah"} ${label.toLowerCase()}`;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
      <div style={ui.header}>
        <div>
          <h1 style={{ ...styles.headingTitle, margin: 0 }}>
            Komunitas & keahlian
          </h1>
          <p style={styles.headingSubtitle}>
            Kelola komunitas, keanggotaan, ketua, dan keahlian pelayanan.
          </p>
        </div>

        <button
          type="button"
          style={styles.btnPrimary}
          disabled={busy}
          onClick={() => openForm("create")}
        >
          + Tambah {label.toLowerCase()}
        </button>
      </div>

      <div style={ui.row}>
        {["komunitas", "keahlian"].map((tab) => (
          <button
            key={tab}
            type="button"
            disabled={busy}
            onClick={() => changeType(tab)}
            style={{
              ...styles.btnSecondary,
              background: type === tab ? "#E0F3F2" : "#fff",
              borderColor: type === tab ? "#00777C" : "#CBD6D6",
            }}
          >
            {tab === "komunitas" ? "Komunitas" : "Keahlian pelayanan"}
          </button>
        ))}
      </div>

      {error && (
        <div role="alert" style={ui.error}>
          {error}
        </div>
      )}
      {notice && (
        <div role="status" style={ui.success}>
          {notice}
        </div>
      )}

      <div style={ui.columns}>
        <section style={{ ...styles.cardBase, minWidth: 0 }}>
          <div style={styles.cardHeader}>
            <div style={styles.cardTitle}>{label}</div>

            <form onSubmit={searchList} style={{ ...ui.row, marginTop: 12 }}>
              <input
                aria-label={`Cari ${label.toLowerCase()}`}
                placeholder={`Cari nama ${label.toLowerCase()}`}
                value={searchInput}
                onChange={(event) => setSearchInput(event.target.value)}
                style={{ ...ui.input, flex: 1 }}
                disabled={busy}
              />
              <button type="submit" style={styles.btnSecondary} disabled={busy}>
                Cari
              </button>
            </form>
          </div>

          {loading ? (
            <div style={styles.emptyContainer}>Memuat data…</div>
          ) : items.length === 0 ? (
            <div style={styles.emptyContainer}>Data tidak ditemukan.</div>
          ) : (
            <div style={{ ...styles.listContainer, maxHeight: 520 }}>
              {items.map((item) => (
                <div
                  key={item.id}
                  style={{
                    ...styles.listItem,
                    background: detail?.id === item.id ? "#F0F9F9" : "#fff",
                  }}
                >
                  <div style={ui.header}>
                    <div>
                      <div style={styles.itemTitle}>
                        {item.nama}{" "}
                        {item.is_sistem && (
                          <span style={styles.badgeSistem}>Sistem</span>
                        )}
                      </div>
                      {item.deskripsi && (
                        <div style={styles.itemDesc}>{item.deskripsi}</div>
                      )}
                    </div>

                    <button
                      type="button"
                      style={styles.btnSecondary}
                      disabled={busy || detailLoading}
                      onClick={() => openDetail(item)}
                    >
                      Detail
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          <div style={ui.pagination}>
            <span style={ui.small}>
              Halaman {meta?.page ?? page} / {meta?.total_pages || 1}
              {" · "}
              {meta?.total ?? 0} hasil
            </span>

            <div style={ui.row}>
              <button
                type="button"
                style={styles.btnSecondary}
                disabled={loading || busy || page <= 1}
                onClick={() => {
                  closeDetail();
                  setPage((value) => value - 1);
                }}
              >
                Sebelumnya
              </button>
              <button
                type="button"
                style={styles.btnSecondary}
                disabled={loading || busy || !meta || page >= meta.total_pages}
                onClick={() => {
                  closeDetail();
                  setPage((value) => value + 1);
                }}
              >
                Berikutnya
              </button>
            </div>
          </div>
        </section>

        {(detail || detailLoading) && (
          <section
            style={{ ...styles.cardBase, flex: "1 1 380px", minWidth: 0 }}
          >
            <div style={styles.cardHeader}>
              <div style={ui.header}>
                <div style={styles.cardTitle}>
                  {detail?.nama || "Memuat detail…"}
                </div>
                <button
                  type="button"
                  style={styles.btnSecondary}
                  disabled={busy}
                  onClick={closeDetail}
                >
                  Tutup
                </button>
              </div>

              {detail?.deskripsi && (
                <p style={styles.itemDesc}>{detail.deskripsi}</p>
              )}

              {detail && (
                <div style={{ ...ui.row, marginTop: 12 }}>
                  <button
                    type="button"
                    style={styles.btnSecondary}
                    disabled={busy || Boolean(detail.is_sistem)}
                    onClick={() => openForm("edit")}
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    style={ui.danger}
                    disabled={busy || Boolean(detail.is_sistem)}
                    onClick={deleteItem}
                  >
                    Hapus
                  </button>
                  <button
                    type="button"
                    style={styles.btnPrimary}
                    disabled={busy}
                    onClick={openMemberModal}
                  >
                    {isGroup ? "+ Tambah anggota" : "+ Beri keahlian"}
                  </button>
                </div>
              )}
            </div>

            {detail && (
              <>
                <div style={{ padding: "12px 20px", ...ui.small }}>
                  {members.length}{" "}
                  {isGroup ? "anggota" : "jemaat pemilik keahlian"}
                </div>

                {members.length === 0 ? (
                  <div style={{ ...styles.emptyContainer, minHeight: 180 }}>
                    {isGroup
                      ? "Belum ada anggota."
                      : "Belum ada jemaat yang memiliki keahlian ini."}
                  </div>
                ) : (
                  <div style={{ ...styles.listContainer, maxHeight: 500 }}>
                    {members.map((member) => (
                      <div key={member.id_user} style={styles.listItem}>
                        <div style={styles.itemTitle}>
                          {member.nama}{" "}
                          {isGroup && member.is_ketua && (
                            <span style={ui.ketuaBadge}>Ketua</span>
                          )}
                        </div>
                        <div style={ui.small}>
                          {member.kode_jemaat || "Belum memiliki kode jemaat"}
                        </div>

                        <div style={{ ...ui.row, marginTop: 8 }}>
                          {isGroup && (
                            <button
                              type="button"
                              style={styles.btnSecondary}
                              disabled={busy}
                              onClick={() => toggleKetua(member)}
                            >
                              {member.is_ketua
                                ? "Batalkan ketua"
                                : "Jadikan ketua"}
                            </button>
                          )}

                          <button
                            type="button"
                            style={ui.danger}
                            disabled={busy}
                            onClick={() => removeMember(member)}
                          >
                            {isGroup ? "Keluarkan" : "Lepas keahlian"}
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </section>
        )}
      </div>

      {modal && (
        <div style={styles.modalOverlay}>
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="komunitas-modal-title"
            style={{
              ...styles.modalBox,
              width: 600,
              maxWidth: "calc(100vw - 40px)",
              maxHeight: "90vh",
              overflowY: "auto",
              boxSizing: "border-box",
            }}
          >
            <h2 id="komunitas-modal-title" style={styles.modalTitle}>
              {title}
            </h2>

            {modalError && (
              <div role="alert" style={{ ...ui.error, marginTop: 12 }}>
                {modalError}
              </div>
            )}

            {modal === "member" ? (
              <div style={ui.modalBody}>
                <form onSubmit={searchUsers} style={ui.row}>
                  <input
                    aria-label="Cari jemaat"
                    placeholder="Nama, kode, email, atau no. HP"
                    value={userInput}
                    onChange={(event) => setUserInput(event.target.value)}
                    style={{ ...ui.input, flex: 1 }}
                    disabled={busy}
                  />
                  <button
                    type="submit"
                    style={styles.btnSecondary}
                    disabled={busy || usersLoading}
                  >
                    Cari
                  </button>
                </form>

                {usersError && (
                  <div role="alert" style={ui.error}>
                    {usersError}
                  </div>
                )}
                <form onSubmit={addMember} style={ui.modalBody}>
                  <fieldset disabled={busy} style={ui.fieldset}>
                    <div style={ui.header}>
                      <label style={ui.row}>
                        <input
                          type="checkbox"
                          checked={allVisibleSelected}
                          disabled={usersLoading || availableUsers.length === 0}
                          onChange={toggleVisibleUsers}
                        />
                        Pilih semua di halaman ini
                      </label>

                      <span style={ui.small}>
                        {selectedUserIDs.length} terpilih
                      </span>
                    </div>

                    {selectedUserIDs.length > 0 && (
                      <button
                        type="button"
                        style={{
                          ...styles.btnSecondary,
                          alignSelf: "flex-start",
                        }}
                        onClick={() => setSelectedUserIDs([])}
                      >
                        Bersihkan pilihan
                      </button>
                    )}

                    <p style={{ ...ui.small, margin: 0 }}>
                      Pilihan tetap tersimpan saat mencari atau berpindah
                      halaman. “Pilih semua” hanya berlaku untuk halaman yang
                      sedang ditampilkan.
                    </p>
                    <div style={ui.userList}>
                      {usersLoading ? (
                        <div style={ui.small}>Memuat jemaat…</div>
                      ) : users.length === 0 ? (
                        <div style={ui.small}>Jemaat tidak ditemukan.</div>
                      ) : (
                        users.map((user) => {
                          const exists = members.some(
                            (member) => member.id_user === user.id,
                          );

                          return (
                            <label
                              key={user.id}
                              style={{
                                ...ui.userOption,
                                opacity: exists ? 0.55 : 1,
                              }}
                            >
                              <input
                                type="checkbox"
                                checked={selectedUserIDs.includes(user.id)}
                                disabled={exists}
                                onChange={() => toggleSelectedUser(user.id)}
                              />
                              <span>
                                <strong>{user.nama}</strong>
                                <span style={{ ...ui.small, display: "block" }}>
                                  {user.kode_jemaat || "Tanpa kode"}
                                  {" · "}
                                  {user.email ||
                                    user.no_telepon ||
                                    "Tanpa kontak"}
                                </span>
                                {exists && (
                                  <span style={ui.small}>
                                    {isGroup
                                      ? "Sudah menjadi anggota"
                                      : "Sudah memiliki keahlian ini"}
                                  </span>
                                )}
                              </span>
                            </label>
                          );
                        })
                      )}
                    </div>

                    <div style={ui.pagination}>
                      <span style={ui.small}>
                        Halaman {userMeta?.page ?? userPage}
                        {" / "}
                        {userMeta?.total_pages || 1}
                      </span>
                      <div style={ui.row}>
                        <button
                          type="button"
                          style={styles.btnSecondary}
                          disabled={usersLoading || userPage <= 1}
                          onClick={() => setUserPage((value) => value - 1)}
                        >
                          Sebelumnya
                        </button>
                        <button
                          type="button"
                          style={styles.btnSecondary}
                          disabled={
                            usersLoading ||
                            !userMeta ||
                            userPage >= userMeta.total_pages
                          }
                          onClick={() => setUserPage((value) => value + 1)}
                        >
                          Berikutnya
                        </button>
                      </div>
                    </div>

                    {isGroup && (
                      <label style={ui.row}>
                        <input
                          type="checkbox"
                          checked={isKetua}
                          onChange={(event) => setIsKetua(event.target.checked)}
                        />
                        Jadikan semua anggota terpilih sebagai ketua
                      </label>
                    )}

                    <div style={ui.modalActions}>
                      <button
                        type="button"
                        style={styles.btnSecondary}
                        onClick={() => setModal(null)}
                      >
                        Batal
                      </button>
                      <button
                        type="submit"
                        style={styles.btnPrimary}
                        disabled={
                          busy || selectedUserIDs.length === 0 || usersLoading
                        }
                      >
                        {busy
                          ? `Menyimpan ${bulkProgress?.completed ?? 0}/${bulkProgress?.total ?? 0}…`
                          : isGroup
                            ? `Tambah ${selectedUserIDs.length} anggota`
                            : `Beri keahlian ke ${selectedUserIDs.length} jemaat`}
                      </button>
                    </div>
                  </fieldset>
                </form>
              </div>
            ) : (
              <form onSubmit={saveForm} style={ui.modalBody}>
                <fieldset disabled={busy} style={ui.fieldset}>
                  <label style={ui.label}>
                    Nama {label.toLowerCase()}
                    <input
                      autoFocus
                      required
                      maxLength={100}
                      value={form.nama}
                      onChange={(event) =>
                        setForm((previous) => ({
                          ...previous,
                          nama: event.target.value,
                        }))
                      }
                      style={ui.input}
                    />
                  </label>

                  {isGroup && (
                    <label style={ui.label}>
                      Deskripsi
                      <textarea
                        rows={4}
                        maxLength={1000}
                        value={form.deskripsi}
                        onChange={(event) =>
                          setForm((previous) => ({
                            ...previous,
                            deskripsi: event.target.value,
                          }))
                        }
                        style={ui.input}
                      />
                    </label>
                  )}

                  <div style={ui.modalActions}>
                    <button
                      type="button"
                      style={styles.btnSecondary}
                      onClick={() => setModal(null)}
                    >
                      Batal
                    </button>
                    <button type="submit" style={styles.btnPrimary}>
                      {busy ? "Menyimpan…" : "Simpan"}
                    </button>
                  </div>
                </fieldset>
              </form>
            )}
          </section>
        </div>
      )}
    </div>
  );
}

const styles = {
  headingTitle: {
    fontSize: 28,
    fontWeight: 800,
    color: "#0D1F21",
    lineHeight: "36px",
  },
  headingSubtitle: {
    color: "#445759",
    fontSize: 14,
    marginTop: 4,
    maxWidth: 680,
  },
  btnPrimary: {
    padding: "9px 16px",
    backgroundColor: "#00777C",
    border: "none",
    borderRadius: 8,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    gap: 8,
    fontSize: 13,
    fontWeight: 600,
    color: "#FFFFFF",
    cursor: "pointer",
    fontFamily: "inherit",
  },
  btnSecondary: {
    padding: "9px 16px",
    backgroundColor: "#FFFFFF",
    border: "1px solid #CBD6D6",
    borderRadius: 8,
    color: "#445759",
    fontSize: 13,
    fontWeight: 600,
    cursor: "pointer",
    fontFamily: "inherit",
  },
  cardBase: {
    flex: "1 1 380px",
    backgroundColor: "#FFFFFF",
    borderRadius: 12,
    border: "1px solid #E1E8E8",
    boxShadow: "0 1px 3px rgba(0,0,0,0.04)",
    overflow: "hidden",
    display: "flex",
    flexDirection: "column",
  },
  cardHeader: {
    padding: "16px 20px",
    borderBottom: "1px solid #E1E8E8",
  },
  cardTitle: {
    color: "#0D1F21",
    fontSize: 16,
    fontWeight: 600,
  },
  emptyContainer: {
    minHeight: 280,
    padding: 20,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    textAlign: "center",
    color: "#66787A",
    fontSize: 13,
  },
  listContainer: {
    display: "flex",
    flexDirection: "column",
    maxHeight: 400,
    overflowY: "auto",
  },
  listItem: {
    padding: "12px 20px",
    borderBottom: "1px solid #E1E8E8",
    display: "flex",
    flexDirection: "column",
    gap: 4,
  },
  itemTitle: {
    color: "#0D1F21",
    fontSize: 14,
    fontWeight: 600,
    overflowWrap: "anywhere",
  },
  itemDesc: {
    color: "#66787A",
    fontSize: 12,
    lineHeight: 1.6,
    whiteSpace: "pre-wrap",
    overflowWrap: "anywhere",
  },
  badgeSistem: {
    padding: "2px 6px",
    backgroundColor: "#EEF3F3",
    color: "#445759",
    fontSize: 11,
    fontWeight: 600,
    borderRadius: 4,
  },
  modalOverlay: {
    position: "fixed",
    inset: 0,
    backgroundColor: "rgba(13,31,33,0.6)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: 9999,
  },
  modalBox: {
    width: 400,
    backgroundColor: "#FFFFFF",
    borderRadius: 12,
    padding: 24,
    boxShadow: "0 12px 32px rgba(0,0,0,0.1)",
  },
  modalTitle: {
    fontSize: 18,
    fontWeight: 700,
    color: "#0D1F21",
    margin: 0,
  },
};

const ui = {
  header: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    flexWrap: "wrap",
    gap: 12,
  },
  row: {
    display: "flex",
    alignItems: "center",
    flexWrap: "wrap",
    gap: 8,
  },
  columns: {
    display: "flex",
    alignItems: "flex-start",
    flexWrap: "wrap",
    gap: 20,
  },
  input: {
    minWidth: 0,
    padding: "10px 12px",
    border: "1px solid #CBD6D6",
    borderRadius: 8,
    fontSize: 14,
    fontFamily: "inherit",
    boxSizing: "border-box",
    background: "#fff",
    color: "#0D1F21",
  },
  small: {
    color: "#66787A",
    fontSize: 12,
  },
  pagination: {
    padding: 12,
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    flexWrap: "wrap",
    gap: 10,
  },
  danger: {
    padding: "8px 12px",
    border: "1px solid #F3B9B9",
    borderRadius: 8,
    background: "#FFF3F3",
    color: "#B42318",
    fontSize: 13,
    fontWeight: 600,
    fontFamily: "inherit",
    cursor: "pointer",
  },
  error: {
    padding: 12,
    borderRadius: 8,
    background: "#FFF0F0",
    color: "#B42318",
    fontSize: 13,
    whiteSpace: "pre-wrap",
  },
  success: {
    padding: 12,
    borderRadius: 8,
    background: "#E0F3E8",
    color: "#09703E",
    fontSize: 13,
  },
  ketuaBadge: {
    background: "#E0F3F2",
    color: "#075E62",
    padding: "2px 7px",
    borderRadius: 5,
    fontSize: 11,
  },
  modalBody: {
    display: "flex",
    flexDirection: "column",
    gap: 16,
    marginTop: 16,
  },
  fieldset: {
    border: 0,
    padding: 0,
    margin: 0,
    minWidth: 0,
    display: "flex",
    flexDirection: "column",
    gap: 16,
  },
  label: {
    display: "flex",
    flexDirection: "column",
    gap: 8,
    color: "#445759",
    fontSize: 13,
    fontWeight: 600,
  },
  userList: {
    display: "flex",
    flexDirection: "column",
    gap: 8,
    maxHeight: 300,
    overflowY: "auto",
  },
  userOption: {
    padding: 12,
    border: "1px solid #E1E8E8",
    borderRadius: 8,
    display: "flex",
    alignItems: "flex-start",
    gap: 10,
    color: "#0D1F21",
    fontSize: 13,
  },
  modalActions: {
    display: "flex",
    justifyContent: "flex-end",
    gap: 10,
  },
};
