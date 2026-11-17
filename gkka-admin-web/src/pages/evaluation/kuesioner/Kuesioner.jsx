import { useEffect, useRef, useState } from "react";
import useKuesioner, { displayTime } from "./useKuesioner";

import fileIcon from "../../../assets/icon-file.svg";
import refreshIcon from "../../../assets/icon-refresh.svg";
import clipboardIcon from "../../../assets/icon-clipboard.svg";
import gridIcon from "../../../assets/icon-grid.svg";
import clockIcon from "../../../assets/icon-clock.svg";
import plusIcon from "../../../assets/icon-plus.svg";
import closeIcon from "../../../assets/icon-x.svg";
import editIcon from "../../../assets/icon-edit.svg";
import uploadIcon from "../../../assets/icon-upload.svg";
import sheetIcon from "../../../assets/icon-sheet.svg";

const questionTypes = [
  ["TEKS_SINGKAT", "Teks singkat"],
  ["PARAGRAF", "Paragraf"],
  ["RATING", "Rating 1–5"],
  ["PILIHAN_GANDA", "Pilihan ganda"],
  ["CENTANG", "Kotak centang"],
  ["DROPDOWN", "Dropdown"],
];

const statusLabels = {
  BELUM_DIBUAT: "Belum dibuat",
  BELUM_DIBUKA: "Terjadwal",
  DIBUKA: "Dibuka",
  DITUTUP: "Ditutup",
};

function Icon({ src }) {
  return <img src={src} alt="" className="ev-icon" />;
}

function Dialog({
  title,
  onClose,
  busy,
  projector = false,
  wide = false,
  children,
}) {
  const root = useRef(null);
  const closeRef = useRef(onClose);
  const busyRef = useRef(busy);

  closeRef.current = onClose;
  busyRef.current = busy;

  useEffect(() => {
    const previous = document.activeElement;
    const oldOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    root.current
      ?.querySelector(
        "input:not(:disabled), button:not(:disabled), select:not(:disabled)",
      )
      ?.focus();

    function handleKey(event) {
      if (event.key === "Escape") {
        event.preventDefault();
        if (!busyRef.current) closeRef.current();
      }

      if (event.key !== "Tab") return;

      const elements = Array.from(
        root.current.querySelectorAll(
          'button, input, select, textarea, a[href], [tabindex="0"]',
        ),
      ).filter(
        (element) => !element.matches(":disabled") && element.offsetParent,
      );

      if (!elements.length) {
        event.preventDefault();
        root.current.focus();
        return;
      }

      const first = elements[0];
      const last = elements[elements.length - 1];

      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    }

    document.addEventListener("keydown", handleKey);

    return () => {
      document.body.style.overflow = oldOverflow;
      document.removeEventListener("keydown", handleKey);
      if (previous?.isConnected) previous.focus();
    };
  }, []);

  return (
    <div className={`ev-overlay ${projector ? "ev-projector-overlay" : ""}`}>
      <section
        ref={root}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        className={
          projector
            ? "ev-projector"
            : `ev-dialog ${wide ? "ev-dialog-wide" : ""}`
        }
      >
        <div className="ev-between">
          <h2>{title}</h2>
          <button
            type="button"
            className="ev-button"
            aria-label="Tutup dialog"
            disabled={busy}
            onClick={onClose}
          >
            <Icon src={closeIcon} />
          </button>
        </div>
        {children}
      </section>
    </div>
  );
}

export default function Kuesioner() {
  const s = useKuesioner();
  const [questionFilter, setQuestionFilter] = useState("aktif");
  const [questionSearch, setQuestionSearch] = useState("");
   const importFileRef = useRef(null);

  const closeDialog = () => {
    if (!s.busy) s.setDialog(null);
  };

  const visibleQuestions = s.questions.filter(
    (item) =>
      (questionFilter === "semua" ||
        (questionFilter === "aktif" ? item.is_aktif : !item.is_aktif)) &&
      item.teks_pertanyaan.toLowerCase().includes(questionSearch.toLowerCase()),
  );

  const activeCount = s.questions.filter((item) => item.is_aktif).length;
  const isSchedule = s.dialog === "schedule" || s.dialog === "annual";
  const hasOptions = ["PILIHAN_GANDA", "CENTANG", "DROPDOWN"].includes(
    s.questionForm?.tipe_jawaban,
  );

  const yearNumber = Number(s.year);
  const validYear =
    Number.isInteger(yearNumber) && yearNumber >= 2000 && yearNumber <= 2100;

  const errorBanner = s.error && (
    <div className="ev-message ev-error" role="alert">
      {s.error}
    </div>
  );

  return (
    <div className="ev-page">
      <style>{css}</style>

      <header className="ev-between">
        <div>
          <h1>Evaluasi kegiatan</h1>
          <p className="ev-muted">
            Kelola jadwal, kode akses, pertanyaan, dan Google Form.
          </p>
        </div>

        <div className="ev-row">
          {["MINGGUAN", "TAHUNAN"].map((type) => (
            <button
              key={type}
              type="button"
              className={`ev-button ${s.jenis === type ? "ev-selected" : ""}`}
              disabled={s.busy}
              aria-pressed={s.jenis === type}
              onClick={() => s.changeJenis(type)}
            >
              {type === "MINGGUAN" ? "Mingguan" : "Tahunan"}
            </button>
          ))}
        </div>
      </header>

      <nav className="ev-tabs" aria-label="Bagian evaluasi">
        {[
          ["operasional", "Jadwal & kode akses"],
          ["pengaturan", "Pertanyaan & template"],
        ].map(([key, label]) => (
          <button
            key={key}
            type="button"
            aria-pressed={s.tab === key}
            className={`ev-button ${s.tab === key ? "ev-selected" : ""}`}
            disabled={s.busy}
            onClick={() => s.setTab(key)}
          >
            {label}
          </button>
        ))}
      </nav>

      {!s.dialog && errorBanner}

      {s.notice && (
        <div className="ev-message ev-success" role="status">
          {s.notice}
        </div>
      )}

      {!s.canEdit && (
        <p className="ev-muted">
          Mode baca. Perubahan hanya dapat dilakukan Super Admin.
        </p>
      )}

      {s.loading ? (
        <section className="ev-card ev-empty" role="status">
          Memuat data evaluasi…
        </section>
      ) : s.tab === "operasional" ? (
        <>
          <section className="ev-card">
            <h2>
              {s.jenis === "MINGGUAN"
                ? "Pilih kegiatan"
                : "Pilih tahun evaluasi"}
            </h2>

            {s.jenis === "MINGGUAN" ? (
              <div className="ev-fields">
                <label>
                  Cari kegiatan
                  <input
                    type="search"
                    value={s.activitySearch}
                    disabled={s.busy}
                    placeholder="Nama kegiatan, tanggal, atau komunitas…"
                    onChange={(event) =>
                      s.setActivitySearch(event.target.value)
                    }
                  />
                </label>

                <label>
                  Kegiatan yang memerlukan evaluasi
                  <select
                    value={s.activityID}
                    disabled={s.busy}
                    onChange={(event) => s.setActivityID(event.target.value)}
                  >
                    <option value="">Pilih kegiatan</option>

                    {s.activityID &&
                      !s.choices.some(
                        (item) => String(item.id) === s.activityID,
                      ) && (
                        <option value={s.activityID}>
                          {s.title} — pilihan saat ini
                        </option>
                      )}

                    {s.choices.map((item) => (
                      <option key={item.id} value={String(item.id)}>
                        {item.tanggal} {item.waktu_mulai}
                        {" · "}
                        {item.nama_kegiatan}
                        {item.historical ? " · arsip kegiatan" : ""}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
            ) : (
              <label className="ev-year">
                Tahun
                <input
                  type="number"
                  min="2000"
                  max="2100"
                  value={s.year}
                  disabled={s.busy}
                  onChange={(event) => s.setYear(event.target.value)}
                />
              </label>
            )}
          </section>

          {(s.jenis === "MINGGUAN" ? Boolean(s.activityID) : validYear) && (
            <div className="ev-columns">
              <section className="ev-card ev-main">
                <div className="ev-between">
                  <h2>{s.title}</h2>
                  <span className={`ev-status ev-status-${s.status}`}>
                    {statusLabels[s.status]}
                  </span>
                </div>

                {s.activity && (
                  <p className="ev-muted">
                    {s.activity.tanggal} · {s.activity.waktu_mulai} WIB
                    {" · "}
                    {s.activity.nama_komunitas || "Umum"}
                  </p>
                )}

                {s.questionnaire ? (
                  <>
                    <div className="ev-code-box">
                      <span className="ev-muted">Kode akses jemaat</span>
                      <div className="ev-code">
                        {s.questionnaire.kode_akses || "Belum tersedia"}
                      </div>
                      <p className="ev-muted">
                        Kode tetap digunakan untuk kuesioner ini.
                      </p>
                    </div>

                    <div className="ev-row">
                      <button
                        type="button"
                        className="ev-button ev-primary"
                        disabled={s.busy || !s.questionnaire.kode_akses}
                        onClick={s.copyCode}
                      >
                        <Icon src={clipboardIcon} /> Salin kode
                      </button>

                      <button
                        type="button"
                        className="ev-button"
                        disabled={s.busy || !s.questionnaire.kode_akses}
                        onClick={() => s.setDialog("projector")}
                      >
                        <Icon src={gridIcon} /> Tampilkan untuk jemaat
                      </button>
                    </div>

                    <div className="ev-divider" />

                    {s.participantLink ? (
                      <a
                        className="ev-button ev-primary"
                        href={s.participantLink}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <Icon src={fileIcon} /> Buka form dengan kode terisi
                      </a>
                    ) : (
                      <button
                        type="button"
                        className="ev-button"
                        disabled={s.busy || s.status !== "DIBUKA"}
                        onClick={s.prepareParticipantLink}
                      >
                        <Icon src={fileIcon} /> Siapkan link pengisian
                      </button>
                    )}

                    <p className="ev-muted">
                      Link pengisian tersedia saat jadwal dibuka dan Google Form
                      menerima jawaban. Kode dan waktu kiriman diperiksa kembali
                      saat respons ditarik.
                    </p>
                  </>
                ) : (
                  <div className="ev-empty">
                    <p>Belum ada kuesioner untuk pilihan ini.</p>

                    {s.canEdit && (
                      <button
                        type="button"
                        className="ev-button ev-primary"
                        disabled={s.busy}
                        onClick={
                          s.jenis === "MINGGUAN"
                            ? s.createWeekly
                            : () => s.openSchedule(true)
                        }
                      >
                        <Icon src={plusIcon} /> Buat kuesioner & kode akses
                      </button>
                    )}
                  </div>
                )}
              </section>

              <section className="ev-card ev-side">
                <h2>
                  <Icon src={clockIcon} /> Jadwal pengisian
                </h2>

                <dl className="ev-details">
                  <div>
                    <dt>Dibuka pada</dt>
                    <dd>{displayTime(s.questionnaire?.dibuka_pada)}</dd>
                  </div>
                  <div>
                    <dt>Ditutup pada</dt>
                    <dd>{displayTime(s.questionnaire?.ditutup_pada)}</dd>
                  </div>
                </dl>

                <p className="ev-muted">
                  Jadwal awal mingguan mengikuti Pengaturan backend. Seluruh
                  waktu di halaman ini memakai WIB.
                </p>

                {s.questionnaire && (
                  <>
                    {s.canEdit && (
                      <div className="ev-row">
                        <button
                          type="button"
                          className={`ev-button ${s.status === "DIBUKA" ? "ev-danger" : "ev-primary"}`}
                          disabled={s.busy}
                          onClick={s.toggleStatus}
                        >
                          {s.status === "DIBUKA"
                            ? "Tutup sekarang"
                            : "Buka sekarang"}
                        </button>

                        <button
                          type="button"
                          className="ev-button"
                          disabled={s.busy}
                          onClick={() => s.openSchedule()}
                        >
                          Ubah jadwal
                        </button>
                      </div>
                    )}

                    <button
                      type="button"
                      className="ev-button"
                      disabled={s.busy}
                      onClick={s.refreshSelected}
                    >
                      <Icon src={refreshIcon} /> Muat ulang status
                    </button>
                  </>
                )}
              </section>
            </div>
          )}
        </>
      ) : (
        <>
          <section className="ev-card">
            <div className="ev-between">
              <div>
                <h2>Pertanyaan Google Form {s.jenis.toLowerCase()}</h2>
                <p className="ev-muted">
                  Pertanyaan dan template digunakan bersama untuk jenis evaluasi
                  ini, bukan hanya kegiatan yang sedang dipilih.
                </p>
              </div>

              {s.canEdit && (
                <button
                  type="button"
                  className="ev-button ev-primary"
                  disabled={s.busy || activeCount === 0}
                  onClick={s.syncGoogle}
                >
                  <Icon src={refreshIcon} />
                  {s.busy ? "Memproses…" : "Kirim ke Google Form"}
                </button>
              )}
            </div>

            {s.syncPending && (
              <div className="ev-message ev-warning">
                Ada perubahan pada sesi ini yang belum dikirim ke Google Form.
              </div>
            )}

            {s.formLink && (
              <a
                className="ev-button"
                href={s.formLink}
                target="_blank"
                rel="noopener noreferrer"
              >
                <Icon src={fileIcon} /> Lihat Google Form hasil sinkronisasi
              </a>
            )}
          </section>

          <div className="ev-columns">
            <section className="ev-card ev-main">
              <div className="ev-between">
                <h2>Daftar pertanyaan · {activeCount} aktif</h2>

                {s.canEdit && (
                  <div className="ev-row">
                    <button
                      type="button"
                      className="ev-button"
                      disabled={s.busy}
                      onClick={() => s.setDialog("import")}
                    >
                      <Icon src={uploadIcon} />
                      Import pertanyaan
                    </button>

                    <button
                      type="button"
                      className="ev-button ev-primary"
                      disabled={s.busy}
                      onClick={() => s.openQuestion()}
                    >
                      <Icon src={plusIcon} />
                      Tambah
                    </button>
                  </div>
                )}
              </div>

              <div className="ev-row">
                <input
                  type="search"
                  aria-label="Cari pertanyaan"
                  placeholder="Cari pertanyaan…"
                  value={questionSearch}
                  onChange={(event) => setQuestionSearch(event.target.value)}
                />
                <select
                  aria-label="Filter status pertanyaan"
                  value={questionFilter}
                  onChange={(event) => setQuestionFilter(event.target.value)}
                >
                  <option value="aktif">Aktif</option>
                  <option value="nonaktif">Nonaktif</option>
                  <option value="semua">Semua</option>
                </select>
              </div>

              {visibleQuestions.length === 0 ? (
                <p className="ev-empty">Tidak ada pertanyaan sesuai filter.</p>
              ) : (
                visibleQuestions.map((item) => (
                  <article key={item.id} className="ev-question">
                    <div className="ev-between">
                      <span className="ev-muted">
                        Urutan {item.urutan} · {item.tipe_jawaban}
                      </span>
                      <span className="ev-muted">
                        {item.is_aktif ? "Aktif" : "Nonaktif"}
                        {item.wajib ? " · Wajib" : ""}
                      </span>
                    </div>

                    <h3>{item.teks_pertanyaan}</h3>

                    {item.opsi?.length > 0 && (
                      <ul>
                        {item.opsi.map((option) => (
                          <li key={option.id}>{option.label}</li>
                        ))}
                      </ul>
                    )}

                    {s.canEdit && (
                      <div className="ev-row">
                        <button
                          type="button"
                          className="ev-button"
                          disabled={s.busy}
                          onClick={() => s.openQuestion(item)}
                        >
                          <Icon src={editIcon} /> Edit
                        </button>
                        <button
                          type="button"
                          className="ev-button"
                          disabled={s.busy}
                          onClick={() => s.toggleQuestion(item)}
                        >
                          {item.is_aktif ? "Nonaktifkan" : "Aktifkan"}
                        </button>
                        <button
                          type="button"
                          className="ev-button ev-danger"
                          disabled={s.busy}
                          onClick={() => s.deleteQuestion(item)}
                        >
                          Hapus
                        </button>
                      </div>
                    )}
                  </article>
                ))
              )}
            </section>

            <div className="ev-side ev-stack">
              <section className="ev-card">
                <h2>Template header</h2>
                <p className="ev-muted">
                  Template tersimpan di backend. Saat disimpan, backend
                  menerapkannya pada kuesioner yang sedang dibuka.
                </p>

                <label>
                  Judul
                  <input
                    value={s.header.judul}
                    disabled={!s.canEdit || s.busy}
                    onChange={(event) =>
                      s.updateHeader("judul", event.target.value)
                    }
                  />
                </label>

                <label>
                  Deskripsi
                  <textarea
                    rows={6}
                    value={s.header.deskripsi}
                    disabled={!s.canEdit || s.busy}
                    onChange={(event) =>
                      s.updateHeader("deskripsi", event.target.value)
                    }
                  />
                </label>

                <div className="ev-tags">
                  {[
                    "nama_kegiatan",
                    "tema",
                    "pembicara",
                    "tanggal",
                    "jam_mulai",
                    "jam_selesai",
                    "komunitas",
                    "nats_alkitab",
                    "ditutup",
                  ].map((tag) => (
                    <code key={tag}>{`{${tag}}`}</code>
                  ))}
                </div>

                <div className="ev-preview">
                  <span className="ev-muted">
                    Preview berdasarkan pilihan saat ini
                  </span>
                  <h3>{s.renderTemplate(s.header.judul)}</h3>
                  <p>{s.renderTemplate(s.header.deskripsi)}</p>
                </div>

                {s.canEdit && (
                  <button
                    type="button"
                    className="ev-button ev-primary"
                    disabled={s.busy || !s.headerDirty}
                    onClick={s.saveHeader}
                  >
                    {s.headerDirty ? "Simpan template" : "Template tersimpan"}
                  </button>
                )}
              </section>
            </div>
          </div>
        </>
      )}

      {s.dialog && (
        <Dialog
          title={
            s.dialog === "import"
              ? "Import pertanyaan"
              : s.dialog === "projector"
                ? "Kode evaluasi"
                : s.dialog === "question"
                  ? s.questionForm?.id
                    ? "Edit pertanyaan"
                    : "Tambah pertanyaan"
                  : s.dialog === "annual"
                    ? `Buat evaluasi tahunan ${s.year}`
                    : "Ubah jadwal pengisian"
          }
          wide={s.dialog === "import"}
          projector={s.dialog === "projector"}
          busy={s.busy}
          onClose={closeDialog}
        >
          {errorBanner}
          {s.dialog === "import" && (
            <div className="ev-import">
              <div className="ev-import-steps" aria-label="Tahap import">
                <span className={!s.preview ? "is-current" : ""}>
                  1. Pilih file
                </span>
                <span aria-hidden="true">→</span>
                <span className={s.preview ? "is-current" : ""}>
                  2. Periksa & simpan
                </span>
              </div>

              <p className="ev-muted">
                Gunakan file .xlsx maksimal 5 MB dengan sheet bernama
                Pertanyaan. File dapat berisi evaluasi Mingguan dan Tahunan.
              </p>

              <input
                ref={importFileRef}
                type="file"
                accept=".xlsx"
                hidden
                disabled={s.busy}
                onChange={s.chooseFile}
              />

              {!s.file ? (
                <button
                  type="button"
                  className="ev-upload-area"
                  disabled={s.busy}
                  onClick={() => importFileRef.current?.click()}
                >
                  <span className="ev-upload-icon">
                    <Icon src={sheetIcon} />
                  </span>

                  <strong>Pilih file Excel pertanyaan</strong>
                  <span>Klik untuk memilih file .xlsx dari komputer</span>

                  <span className="ev-button ev-primary">
                    <Icon src={uploadIcon} />
                    Pilih file
                  </span>
                </button>
              ) : (
                <div className="ev-import-file">
                  <span className="ev-upload-icon">
                    <Icon src={sheetIcon} />
                  </span>

                  <div className="ev-import-file-info">
                    <strong>{s.file.name}</strong>
                    <span className="ev-muted">
                      {(s.file.size / 1024).toLocaleString("id-ID", {
                        maximumFractionDigits: 1,
                      })}{" "}
                      KB · Excel
                    </span>
                  </div>

                  <button
                    type="button"
                    className="ev-button"
                    disabled={s.busy}
                    onClick={() => importFileRef.current?.click()}
                  >
                    Ganti file
                  </button>
                </div>
              )}

              {s.busy && (
                <div className="ev-import-loading" role="status">
                  <span className="ev-spinner" aria-hidden="true" />
                  Memproses file, mohon tunggu…
                </div>
              )}

              {!s.busy && s.file && !s.preview && (
                <button
                  type="button"
                  className="ev-button"
                  onClick={s.previewImport}
                >
                  <Icon src={refreshIcon} />
                  Periksa ulang file
                </button>
              )}

              {s.preview && (
                <>
                  <div className="ev-import-stats">
                    <div>
                      <strong>{s.preview.total_baris}</strong>
                      <span>Total baris</span>
                    </div>
                    <div className="ev-stat-valid">
                      <strong>{s.preview.baris_valid}</strong>
                      <span>Valid</span>
                    </div>
                    <div
                      className={
                        s.preview.baris_error > 0 ? "ev-stat-error" : ""
                      }
                    >
                      <strong>{s.preview.baris_error}</strong>
                      <span>Perlu diperbaiki</span>
                    </div>
                  </div>

                  {s.preview.baris_error > 0 && (
                    <div className="ev-message ev-warning">
                      Perbaiki baris yang bermasalah di Excel, lalu pilih
                      kembali file yang sudah diperbaiki. Belum ada perubahan
                      yang disimpan.
                    </div>
                  )}

                  <div
                    className="ev-import-table-wrap"
                    tabIndex={0}
                    role="region"
                    aria-label="Preview pertanyaan"
                  >
                    <table className="ev-import-table">
                      <thead>
                        <tr>
                          <th>Baris</th>
                          <th>Jenis</th>
                          <th>Pertanyaan</th>
                          <th>Hasil pemeriksaan</th>
                        </tr>
                      </thead>

                      <tbody>
                        {(s.preview.baris || []).map((row) => {
                          const hasError =
                            row.errors && Object.keys(row.errors).length > 0;

                          return (
                            <tr
                              key={row.baris}
                              className={hasError ? "ev-import-invalid" : ""}
                            >
                              <td>{row.baris}</td>
                              <td>{row.jenis_kuesioner || "—"}</td>
                              <td>
                                <strong>{row.teks_pertanyaan || "—"}</strong>
                                <div className="ev-muted">
                                  {row.tipe_jawaban}
                                  {" · "}Urutan {row.urutan}
                                  {row.wajib ? " · Wajib" : ""}
                                </div>

                                {row.opsi?.length > 0 && (
                                  <div className="ev-muted">
                                    Pilihan: {row.opsi.join("; ")}
                                  </div>
                                )}
                              </td>
                              <td>
                                {hasError ? (
                                  <ul className="ev-import-errors">
                                    {Object.entries(row.errors).map(
                                      ([field, text]) => (
                                        <li key={field}>
                                          <strong>{field}:</strong> {text}
                                        </li>
                                      ),
                                    )}
                                  </ul>
                                ) : (
                                  <span className="ev-import-valid-label">
                                    {row.status === "SAMA"
                                      ? "Valid · gunakan pertanyaan lama"
                                      : "Valid · pertanyaan baru"}
                                  </span>
                                )}
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>

                  {(s.preview.akan_dinonaktifkan || []).length > 0 && (
                    <details className="ev-import-deactivate">
                      <summary>
                        {s.preview.akan_dinonaktifkan.length} pertanyaan lama
                        akan dinonaktifkan
                      </summary>

                      <ul>
                        {s.preview.akan_dinonaktifkan.map((item) => (
                          <li key={item.id}>
                            <strong>{item.jenis_kuesioner}:</strong>{" "}
                            {item.teks_pertanyaan}
                          </li>
                        ))}
                      </ul>
                    </details>
                  )}
                </>
              )}

              <div className="ev-import-footer">
                <span className="ev-muted">
                  {s.preview?.bisa_disimpan
                    ? "Preview valid. Siap disimpan ke database."
                    : "Penyimpanan aktif setelah file lolos pemeriksaan."}
                </span>

                <div className="ev-actions">
                  <button
                    type="button"
                    className="ev-button"
                    disabled={s.busy}
                    onClick={closeDialog}
                  >
                    Batal
                  </button>

                  <button
                    type="button"
                    className="ev-button ev-primary"
                    disabled={s.busy || !s.preview?.bisa_disimpan}
                    onClick={s.saveImport}
                  >
                    {s.busy ? "Memproses…" : "Simpan import"}
                  </button>
                </div>
              </div>
            </div>
          )}

          {s.dialog === "projector" && (
            <div className="ev-projector-content">
              <p>{s.title}</p>
              <span>Masukkan kode berikut di aplikasi jemaat</span>
              <div className="ev-projector-code">
                {s.questionnaire?.kode_akses}
              </div>
              <strong>{statusLabels[s.status]}</strong>
              <p>
                Batas pengisian: {displayTime(s.questionnaire?.ditutup_pada)}
              </p>
              <small>Tekan Esc untuk kembali</small>
            </div>
          )}

          {isSchedule && (
            <form onSubmit={s.saveSchedule}>
              <fieldset disabled={s.busy}>
                <p className="ev-muted">
                  Gunakan waktu Indonesia Barat (WIB), termasuk jika komputer
                  berada di zona waktu lain.
                </p>

                {["dibuka_pada", "ditutup_pada"].map((key) => (
                  <label key={key}>
                    {key === "dibuka_pada" ? "Dibuka pada" : "Ditutup pada"}
                    <input
                      type="datetime-local"
                      required
                      value={s.schedule[key]}
                      onChange={(event) =>
                        s.setSchedule((previous) => ({
                          ...previous,
                          [key]: event.target.value,
                        }))
                      }
                    />
                  </label>
                ))}

                <div className="ev-actions">
                  <button
                    type="button"
                    className="ev-button"
                    onClick={closeDialog}
                  >
                    Batal
                  </button>
                  <button type="submit" className="ev-button ev-primary">
                    {s.busy ? "Menyimpan…" : "Simpan"}
                  </button>
                </div>
              </fieldset>
            </form>
          )}

          {s.dialog === "question" && s.questionForm && (
            <form onSubmit={s.saveQuestion}>
              <fieldset disabled={s.busy}>
                <label>
                  Teks pertanyaan
                  <textarea
                    required
                    rows={3}
                    value={s.questionForm.teks_pertanyaan}
                    onChange={(event) =>
                      s.setQuestionForm((previous) => ({
                        ...previous,
                        teks_pertanyaan: event.target.value,
                      }))
                    }
                  />
                </label>

                <div className="ev-fields">
                  <label>
                    Tipe jawaban
                    <select
                      value={s.questionForm.tipe_jawaban}
                      onChange={(event) =>
                        s.setQuestionForm((previous) => ({
                          ...previous,
                          tipe_jawaban: event.target.value,
                        }))
                      }
                    >
                      {questionTypes.map(([key, label]) => (
                        <option key={key} value={key}>
                          {label}
                        </option>
                      ))}
                    </select>
                  </label>

                  <label>
                    Urutan
                    <input
                      type="number"
                      min="1"
                      step="1"
                      required
                      value={s.questionForm.urutan}
                      onChange={(event) =>
                        s.setQuestionForm((previous) => ({
                          ...previous,
                          urutan: event.target.value,
                        }))
                      }
                    />
                  </label>
                </div>

                <label className="ev-checkbox">
                  <input
                    type="checkbox"
                    checked={s.questionForm.wajib}
                    onChange={(event) =>
                      s.setQuestionForm((previous) => ({
                        ...previous,
                        wajib: event.target.checked,
                      }))
                    }
                  />
                  Wajib diisi
                </label>

                {hasOptions && (
                  <label>
                    Pilihan jawaban, pisahkan dengan titik koma (;)
                    <textarea
                      required
                      rows={3}
                      value={s.questionForm.opsiText}
                      onChange={(event) =>
                        s.setQuestionForm((previous) => ({
                          ...previous,
                          opsiText: event.target.value,
                        }))
                      }
                      placeholder="Sangat baik; Baik; Cukup"
                    />
                  </label>
                )}

                <div className="ev-actions">
                  <button
                    type="button"
                    className="ev-button"
                    onClick={closeDialog}
                  >
                    Batal
                  </button>
                  <button type="submit" className="ev-button ev-primary">
                    {s.busy ? "Menyimpan…" : "Simpan pertanyaan"}
                  </button>
                </div>
              </fieldset>
            </form>
          )}
        </Dialog>
      )}
    </div>
  );
}

const css = `
.ev-page{display:flex;flex-direction:column;gap:20px;color:#0D1F21}
.ev-page *{box-sizing:border-box}
.ev-page h1{margin:0;font-size:28px}
.ev-page h2{margin:0;font-size:17px;display:flex;align-items:center;gap:8px}
.ev-page h3{font-size:14px;margin:10px 0}
.ev-page p{line-height:1.6}
.ev-page button,.ev-page input,.ev-page select,.ev-page textarea{font:inherit}
.ev-page button:disabled{opacity:.5;cursor:not-allowed}
.ev-page button:focus-visible,.ev-page a:focus-visible,
.ev-page input:focus-visible,.ev-page select:focus-visible,
.ev-page textarea:focus-visible{outline:3px solid #C18B16;outline-offset:3px}
.ev-row,.ev-between{display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.ev-between{justify-content:space-between}
.ev-muted{color:#66787A;font-size:12px;line-height:1.6}
.ev-button{display:inline-flex;align-items:center;justify-content:center;gap:8px;
padding:9px 13px;border:1px solid #CBD6D6;border-radius:8px;background:white;
color:#0D1F21;cursor:pointer;text-decoration:none;font-size:13px;font-weight:600}
.ev-primary{background:#00777C;color:white;border-color:#00777C}
.ev-primary .ev-icon{filter:brightness(0) invert(1)}
.ev-danger{background:#FFF3F3;border-color:#F3B9B9;color:#B42318}
.ev-selected{background:#E0F3F2;border-color:#00777C;color:#075E62}
.ev-icon{width:17px;height:17px;flex-shrink:0}
.ev-tabs{display:flex;gap:8px;padding-bottom:12px;border-bottom:1px solid #E1E8E8;flex-wrap:wrap}
.ev-card{background:white;padding:22px;border:1px solid #E1E8E8;border-radius:14px;
display:flex;flex-direction:column;gap:16px;min-width:0}
.ev-columns{display:flex;gap:20px;align-items:flex-start;flex-wrap:wrap}
.ev-main{flex:2 1 420px}
.ev-side{flex:1 1 320px;min-width:0}
.ev-stack{display:flex;flex-direction:column;gap:20px}
.ev-fields{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px}
.ev-page label{display:flex;flex-direction:column;gap:7px;font-size:13px;font-weight:600;color:#445759}
.ev-page input:not([type=checkbox]),.ev-page select,.ev-page textarea{
padding:10px 12px;border:1px solid #CBD6D6;border-radius:8px;min-width:0;
background:white;color:#0D1F21;max-width:100%;font-size:14px}
.ev-page textarea{resize:vertical}
.ev-page .ev-checkbox{flex-direction:row;align-items:center}
.ev-year{max-width:200px}
.ev-message{padding:13px 16px;border-radius:8px;font-size:13px;white-space:pre-wrap;line-height:1.6}
.ev-error{background:#FFF0F0;color:#B42318}
.ev-success{background:#E0F3E8;color:#09703E}
.ev-warning{background:#FFF4DE;color:#92400E}
.ev-status{padding:5px 10px;border-radius:20px;font-size:11px;font-weight:700;background:#EEF3F3;color:#445759}
.ev-status-DIBUKA{background:#E0F3E8;color:#09703E}
.ev-status-BELUM_DIBUKA{background:#FFF4DE;color:#92400E}
.ev-status-DITUTUP{background:#FFF0F0;color:#B42318}
.ev-code-box{background:#F0F9F8;border:1px solid #CFE9E6;padding:24px;text-align:center;border-radius:12px}
.ev-code{font-family:monospace;font-size:clamp(32px,5vw,56px);letter-spacing:.16em;
font-weight:800;color:#075E62;user-select:all;margin:14px 0;overflow-wrap:anywhere}
.ev-empty{text-align:center;padding:30px 10px;color:#66787A}
.ev-divider{height:1px;background:#E1E8E8}
.ev-details{margin:0;display:grid;gap:18px}
.ev-details dt{font-size:12px;color:#66787A}
.ev-details dd{margin:5px 0 0;font-size:14px;font-weight:600}
.ev-question{padding:16px;border:1px solid #E1E8E8;border-radius:10px;background:#FAFCFC}
.ev-question ul{font-size:13px;color:#445759;padding-left:20px}
.ev-tags{display:flex;flex-wrap:wrap;gap:5px}
.ev-tags code{font-size:10px;background:#EEF3F3;color:#075E62;padding:4px 6px;border-radius:4px}
.ev-preview{background:#F7FAFA;border:1px solid #E1E8E8;padding:14px;border-radius:8px}
.ev-preview p{white-space:pre-wrap;font-size:13px}
.ev-import-list{max-height:260px;overflow:auto}
.ev-import-row{padding:10px 0;border-bottom:1px solid #E1E8E8;font-size:12px}
.ev-overlay{position:fixed;inset:0;background:rgba(13,31,33,.65);z-index:9999;
display:flex;align-items:center;justify-content:center;padding:20px}
.ev-dialog{width:600px;max-width:100%;max-height:90vh;overflow:auto;background:white;
border-radius:14px;padding:24px;display:flex;flex-direction:column;gap:18px}
.ev-page fieldset{border:0;padding:0;margin:0;min-width:0;display:flex;flex-direction:column;gap:18px}
.ev-actions{display:flex;justify-content:flex-end;gap:10px}
.ev-projector-overlay{padding:0;background:#0D2B2E}
.ev-projector{width:100%;height:100%;overflow:auto;padding:28px;color:white;display:flex;flex-direction:column}
.ev-projector h2{color:#C8DFDE}
.ev-projector-content{flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;gap:16px}
.ev-projector-content>p:first-child{font-size:clamp(20px,3vw,36px)}
.ev-projector-code{font-family:monospace;font-size:clamp(48px,12vw,180px);
font-weight:800;letter-spacing:.12em;color:#F3D88A;user-select:all;overflow-wrap:anywhere;max-width:100%}
@media(max-width:700px){
.ev-main,.ev-side{flex-basis:100%}
.ev-card{padding:16px}
.ev-page h1{font-size:24px}
.ev-dialog{padding:18px}
}
.ev-dialog-wide{
  width:960px;
}
.ev-import{
  display:flex;
  flex-direction:column;
  gap:16px;
}
.ev-import-steps{
  display:flex;
  align-items:center;
  gap:12px;
  color:#829194;
  font-size:12px;
}
.ev-import-steps .is-current{
  color:#075E62;
  font-weight:700;
}
.ev-upload-area{
  display:flex;
  flex-direction:column;
  align-items:center;
  justify-content:center;
  gap:12px;
  width:100%;
  min-height:220px;
  padding:28px;
  border:2px dashed #A9CFCD;
  border-radius:14px;
  background:#F3FAF9;
  color:#075E62;
  cursor:pointer;
  text-align:center;
}
.ev-upload-area:hover{
  background:#EAF6F4;
  border-color:#00777C;
}
.ev-upload-area>span:not(.ev-upload-icon):not(.ev-button){
  font-size:12px;
  color:#66787A;
}
.ev-upload-icon{
  display:grid;
  place-items:center;
  width:48px;
  height:48px;
  border-radius:12px;
  background:#E0F3F2;
  flex-shrink:0;
}
.ev-upload-icon .ev-icon{
  width:25px;
  height:25px;
}
.ev-import-file{
  display:flex;
  align-items:center;
  gap:14px;
  padding:14px;
  border:1px solid #D9E7E5;
  border-radius:12px;
  background:#F7FAFA;
}
.ev-import-file-info{
  flex:1;
  min-width:0;
  display:flex;
  flex-direction:column;
  gap:5px;
}
.ev-import-file-info strong{
  overflow-wrap:anywhere;
  font-size:14px;
}
.ev-import-stats{
  display:grid;
  grid-template-columns:repeat(3,minmax(0,1fr));
  gap:12px;
}
.ev-import-stats>div{
  display:flex;
  flex-direction:column;
  gap:5px;
  padding:16px;
  background:#F5F8F8;
  border:1px solid #E1E8E8;
  border-radius:10px;
}
.ev-import-stats strong{
  font-size:27px;
}
.ev-import-stats span{
  font-size:12px;
  color:#66787A;
}
.ev-import-stats .ev-stat-valid{
  background:#EDF8F1;
  color:#09703E;
  border-color:#CDE8D6;
}
.ev-import-stats .ev-stat-error{
  background:#FFF1F1;
  color:#B42318;
  border-color:#F4D0D0;
}
.ev-import-table-wrap{
  max-height:330px;
  overflow:auto;
  border:1px solid #E1E8E8;
  border-radius:10px;
}
.ev-import-table{
  width:100%;
  min-width:650px;
  border-collapse:collapse;
  text-align:left;
  font-size:12px;
}
.ev-import-table th{
  position:sticky;
  top:0;
  z-index:1;
  padding:12px;
  background:#EEF5F4;
  color:#445759;
  font-weight:600;
}
.ev-import-table td{
  padding:12px;
  vertical-align:top;
  border-top:1px solid #E1E8E8;
  line-height:1.6;
}
.ev-import-table th:first-child{
  width:65px;
}
.ev-import-table th:nth-child(2){
  width:110px;
}
.ev-import-table th:last-child{
  width:240px;
}
.ev-import-invalid{
  background:#FFF7F7;
}
.ev-import-errors{
  margin:0;
  padding-left:16px;
  color:#B42318;
}
.ev-import-valid-label{
  color:#09703E;
}
.ev-import-deactivate{
  padding:14px;
  background:#FFF8E8;
  border:1px solid #EEDDB4;
  border-radius:10px;
  color:#92400E;
  font-size:12px;
  line-height:1.7;
}
.ev-import-deactivate summary{
  cursor:pointer;
  font-weight:600;
}
.ev-import-footer{
  display:flex;
  justify-content:space-between;
  align-items:center;
  flex-wrap:wrap;
  gap:14px;
  padding-top:16px;
  border-top:1px solid #E1E8E8;
}
.ev-import-loading{
  display:flex;
  align-items:center;
  justify-content:center;
  gap:10px;
  padding:20px;
  color:#075E62;
  font-size:13px;
}
.ev-spinner{
  width:18px;
  height:18px;
  border:2px solid #CFE4E2;
  border-top-color:#00777C;
  border-radius:50%;
  animation:ev-spin .8s linear infinite;
}
@keyframes ev-spin{
  to{transform:rotate(360deg)}
}
@media(prefers-reduced-motion:reduce){
  .ev-spinner{animation:none}
}
@media(max-width:600px){
  .ev-import-file{flex-wrap:wrap}
  .ev-import-stats{gap:6px}
  .ev-import-stats>div{padding:12px 8px}
  .ev-import-footer .ev-actions{width:100%}
  .ev-import-footer .ev-actions button{flex:1}
}
`;
