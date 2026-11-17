import { useEffect, useRef, useState } from "react";
import api from "../../../services/api";
import { getSession } from "../../../services/session";

const optionsTypes = ["PILIHAN_GANDA", "CENTANG", "DROPDOWN"];

const defaults = {
  MINGGUAN: {
    judul: "Evaluasi {nama_kegiatan}",
    deskripsi:
      "Tema: {tema}\nPembicara: {pembicara}\n{tanggal} pukul {jam_mulai}\n\nForm dapat diisi sampai {ditutup}.",
  },
  TAHUNAN: {
    judul: "{nama_kegiatan}",
    deskripsi: "Evaluasi tahunan.\nForm dapat diisi sampai {ditutup}.",
  },
};

export function displayTime(value) {
  if (!value) return "—";

  return (
    new Date(value).toLocaleString("id-ID", {
      timeZone: "Asia/Jakarta",
      dateStyle: "medium",
      timeStyle: "short",
    }) + " WIB"
  );
}

export function inputTime(value) {
  if (!value) return "";

  const parts = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Asia/Jakarta",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  }).formatToParts(new Date(value));

  const get = (type) => parts.find((part) => part.type === type)?.value;

  return `${get("year")}-${get("month")}-${get("day")}T${get("hour")}:${get("minute")}`;
}

function sqlTime(value) {
  return value ? value.replace("T", " ") + ":00" : "";
}

function errorText(error) {
  return [
    error.response?.data?.message || error.message || "Permintaan gagal.",
    ...Object.entries(error.response?.data?.errors || {}).map(
      ([field, value]) => `${field}: ${value}`,
    ),
  ].join("\n");
}

async function allActivities(signal) {
  const items = [];
  let page = 1;

  while (true) {
    const response = await api.get("/kegiatan", {
      params: { page, limit: 100 },
      signal,
    });

    if (signal.aborted) return [];

    items.push(...(response.data.data.items || []));

    if (page >= (response.data.data.meta?.total_pages || 1)) return items;
    page += 1;
  }
}

export default function useKuesioner() {
  const canEdit = getSession()?.user?.role === "Super Admin";

  const [jenis, setJenis] = useState("MINGGUAN");
  const [tab, setTab] = useState("operasional");
  const [activities, setActivities] = useState([]);
  const [questionnaires, setQuestionnaires] = useState([]);
  const [activityID, setActivityID] = useState("");
  const [year, setYear] = useState(String(new Date().getFullYear()));
  const [activitySearch, setActivitySearch] = useState("");
  const [questions, setQuestions] = useState([]);
  const [header, setHeader] = useState({ ...defaults.MINGGUAN });
  const [headerDirty, setHeaderDirty] = useState(false);
  const [syncDirty, setSyncDirty] = useState({});
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [formLink, setFormLink] = useState("");
  const [participantLink, setParticipantLink] = useState("");
  const [now, setNow] = useState(Date.now());

  const [dialog, setDialog] = useState(null);
  const [schedule, setSchedule] = useState({
    dibuka_pada: "",
    ditutup_pada: "",
  });
  const [questionForm, setQuestionForm] = useState(null);

  const [file, setFile] = useState(null);
  const [preview, setPreview] = useState(null);
  const [fileVersion, setFileVersion] = useState(0);

  const lock = useRef(false);
  const context = useRef(0);

  const questionnaire =
    questionnaires.find((item) =>
      jenis === "MINGGUAN"
        ? String(item.id_kegiatan) === activityID
        : String(item.tahun) === year,
    ) || null;

  const activity =
    activities.find((item) => String(item.id) === activityID) || null;

  const activityChoices = activities.filter(
    (item) =>
      item.perlu_evaluasi ||
      questionnaires.some((q) => q.id_kegiatan === item.id),
  );

  const historicalChoices = questionnaires
    .filter(
      (q) =>
        q.id_kegiatan &&
        !activityChoices.some((item) => item.id === q.id_kegiatan),
    )
    .map((q) => ({
      id: q.id_kegiatan,
      nama_kegiatan: q.judul_form,
      tanggal: "",
      waktu_mulai: "",
      historical: true,
    }));

  const choices = [...activityChoices, ...historicalChoices].filter((item) =>
    `${item.nama_kegiatan} ${item.tanggal} ${item.nama_komunitas || ""}`
      .toLowerCase()
      .includes(activitySearch.trim().toLowerCase()),
  );

  const title =
    jenis === "MINGGUAN"
      ? activity?.nama_kegiatan || questionnaire?.judul_form || "Pilih kegiatan"
      : `Evaluasi Tahunan ${year}`;

  let status = "BELUM_DIBUAT";

  if (questionnaire) {
    const start = questionnaire.dibuka_pada
      ? new Date(questionnaire.dibuka_pada).getTime()
      : null;
    const end = questionnaire.ditutup_pada
      ? new Date(questionnaire.ditutup_pada).getTime()
      : null;

    status =
      end !== null && now >= end
        ? "DITUTUP"
        : start !== null && now < start
          ? "BELUM_DIBUKA"
          : "DIBUKA";
  }

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 15000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    const version = ++context.current;

    setLoading(true);
    setError("");
    setNotice("");
    setQuestionnaires([]);
    setQuestions([]);
    setActivities([]);
    setActivityID("");
    setHeader({ ...defaults[jenis] });
    setHeaderDirty(false);
    setFormLink("");
    setParticipantLink("");
    setFile(null);
    setPreview(null);

    async function load() {
      const results = await Promise.allSettled([
        api.get("/kuesioner", {
          params: { jenis },
          signal: controller.signal,
        }),
        api.get("/pertanyaan", {
          params: { jenis },
          signal: controller.signal,
        }),
        api.get("/pertanyaan/header", {
          params: { jenis },
          signal: controller.signal,
        }),
        jenis === "MINGGUAN"
          ? allActivities(controller.signal)
          : Promise.resolve([]),
      ]);

      if (controller.signal.aborted || context.current !== version) return;

      const failures = [];

      if (results[0].status === "fulfilled") {
        setQuestionnaires(results[0].value.data.data || []);
      } else {
        failures.push(`Kuesioner: ${errorText(results[0].reason)}`);
      }

      if (results[1].status === "fulfilled") {
        setQuestions(results[1].value.data.data || []);
      } else {
        failures.push(`Pertanyaan: ${errorText(results[1].reason)}`);
      }

      if (results[2].status === "fulfilled") {
        const saved = results[2].value.data.data;

        setHeader({
          judul: saved?.judul || defaults[jenis].judul,
          deskripsi: saved?.judul
            ? saved.deskripsi || ""
            : defaults[jenis].deskripsi,
        });
      } else {
        failures.push(`Template: ${errorText(results[2].reason)}`);
      }

      if (results[3].status === "fulfilled") {
        const items = results[3].value;
        setActivities(items);

        const eligible = items.filter((item) => item.perlu_evaluasi);
        const today = inputTime(new Date().toISOString()).slice(0, 10);
        const next =
          eligible.find((item) => item.tanggal >= today) ||
          eligible[eligible.length - 1];

        setActivityID(next ? String(next.id) : "");
      } else {
        failures.push(`Kegiatan: ${errorText(results[3].reason)}`);
      }

      setError(failures.join("\n"));
      setLoading(false);
    }

    load();
    return () => controller.abort();
  }, [jenis]);

  useEffect(() => {
    setParticipantLink("");
  }, [activityID, year, jenis, status]);

  function changeJenis(value) {
    if (lock.current || value === jenis) return;

    if (
      headerDirty &&
      !window.confirm("Template belum disimpan. Pindah jenis evaluasi?")
    )
      return;

    setDialog(null);
    setJenis(value);
  }

  async function run(task, adminOnly = true) {
    if (lock.current || (adminOnly && !canEdit)) return;

    lock.current = true;
    setBusy(true);
    setError("");
    setNotice("");

    try {
      await task();
    } catch (err) {
      setError(errorText(err));
    } finally {
      lock.current = false;
      setBusy(false);
      setNow(Date.now());
    }
  }

  function upsertQuestionnaire(item) {
    setQuestionnaires((previous) => [
      item,
      ...previous.filter((q) => q.id !== item.id),
    ]);
  }

  async function refreshQuestionnaires() {
    const response = await api.get("/kuesioner", { params: { jenis } });
    const items = response.data.data || [];
    setQuestionnaires(items);
    return items;
  }

  async function refreshQuestions() {
    const response = await api.get("/pertanyaan", { params: { jenis } });
    setQuestions(response.data.data || []);
  }

  function createWeekly() {
    return run(async () => {
      if (!activityID || !activity?.perlu_evaluasi) {
        throw new Error("Pilih kegiatan yang memerlukan evaluasi.");
      }

      // Periksa ulang agar kuesioner hasil import tidak dibuat ganda.
      const latest = await refreshQuestionnaires();
      const existing = latest.find(
        (item) => String(item.id_kegiatan) === activityID,
      );

      if (existing) {
        setNotice("Kuesioner sudah tersedia. Kode akses yang ada digunakan.");
        return;
      }

      try {
        const response = await api.post("/kuesioner", {
          jenis_kuesioner: "MINGGUAN",
          id_kegiatan: Number(activityID),
        });

        upsertQuestionnaire(response.data.data);
        setNotice("Kuesioner dan kode akses berhasil dibuat.");
      } catch (err) {
        if (err.response?.status === 409) {
          const items = await refreshQuestionnaires();
          if (items.some((item) => String(item.id_kegiatan) === activityID)) {
            setNotice("Kuesioner sudah dibuat. Kode akses berhasil dimuat.");
            return;
          }
        }
        throw err;
      }
    });
  }

  function openSchedule(createAnnual = false) {
    setError("");
    setSchedule({
      dibuka_pada: inputTime(questionnaire?.dibuka_pada),
      ditutup_pada: inputTime(questionnaire?.ditutup_pada),
    });
    setDialog(createAnnual ? "annual" : "schedule");
  }

  function saveSchedule(event) {
    event.preventDefault();

    return run(async () => {
      if (
        !schedule.dibuka_pada ||
        !schedule.ditutup_pada ||
        schedule.ditutup_pada <= schedule.dibuka_pada
      ) {
        throw new Error(
          "Jam tutup harus setelah jam buka. Semua waktu memakai WIB.",
        );
      }

      const payload = {
        dibuka_pada: sqlTime(schedule.dibuka_pada),
        ditutup_pada: sqlTime(schedule.ditutup_pada),
      };

      if (dialog === "annual") {
        const latest = await refreshQuestionnaires();
        const existing = latest.find((item) => String(item.tahun) === year);

        if (existing) {
          setDialog(null);
          setNotice("Kuesioner tahun ini sudah tersedia.");
          return;
        }

        const response = await api.post("/kuesioner", {
          ...payload,
          jenis_kuesioner: "TAHUNAN",
          tahun: Number(year),
        });

        upsertQuestionnaire(response.data.data);
      } else {
        if (!questionnaire) throw new Error("Pilih kuesioner terlebih dahulu.");

        const response = await api.put(
          `/kuesioner/${questionnaire.id}/jadwal`,
          payload,
        );

        upsertQuestionnaire(response.data.data);
      }

      setDialog(null);
      setNotice("Jadwal berhasil disimpan.");
    });
  }

  function toggleStatus() {
    return run(async () => {
      if (!questionnaire) return;

      const response = await api.put(`/kuesioner/${questionnaire.id}/jadwal`, {
        status: status === "DIBUKA" ? "TUTUP" : "BUKA",
      });

      upsertQuestionnaire(response.data.data);
      setNotice("Status dan jadwal kuesioner berhasil diperbarui.");
    });
  }

  function copyCode() {
    return run(async () => {
      if (!questionnaire?.kode_akses) return;

      try {
        await navigator.clipboard.writeText(questionnaire.kode_akses);
        setNotice("Kode akses berhasil disalin.");
      } catch {
        throw new Error(
          "Tidak dapat menyalin otomatis. Blok kode yang tampil lalu tekan Ctrl+C.",
        );
      }
    }, false);
  }

  function prepareParticipantLink() {
    return run(async () => {
      if (!questionnaire?.kode_akses) return;

      const response = await api.post("/evaluasi/kode", {
        kode: questionnaire.kode_akses,
      });

      const data = response.data.data;

      if (data.id_kuesioner !== questionnaire.id || !data.link_form) {
        throw new Error("Link tidak cocok dengan kuesioner yang dipilih.");
      }

      setParticipantLink(data.link_form);
      setNotice("Link dengan kode terisi sudah siap dibuka.");
    }, false);
  }

  function saveHeader() {
    return run(async () => {
      if (!header.judul.trim()) throw new Error("Judul template wajib diisi.");

      await api.put("/pertanyaan/header", {
        jenis_kuesioner: jenis,
        judul: header.judul,
        deskripsi: header.deskripsi,
      });

      setHeaderDirty(false);
      setNotice(
        "Template tersimpan. Backend menerapkannya ke kuesioner yang sedang dibuka.",
      );
    });
  }

  function updateHeader(key, value) {
    setHeader((previous) => ({ ...previous, [key]: value }));
    setHeaderDirty(true);
  }

  function syncGoogle() {
    return run(async () => {
      const response = await api.post("/pertanyaan/gform", null, {
        params: { jenis },
      });

      const data = response.data.data;
      setFormLink(data.link_form || "");
      setSyncDirty((previous) => ({ ...previous, [jenis]: false }));
      setNotice(
        `Sinkron selesai: ${data.ditambah} ditambah, ${data.diubah} diubah, ${data.dipindah} dipindah, ${data.dihapus} dihapus.`,
      );
    });
  }

  function openQuestion(item = null) {
    setError("");
    setQuestionForm({
      id: item?.id,
      teks_pertanyaan: item?.teks_pertanyaan || "",
      tipe_jawaban: item?.tipe_jawaban || "TEKS_SINGKAT",
      urutan:
        item?.urutan || Math.max(0, ...questions.map((q) => q.urutan)) + 1,
      wajib: item ? item.wajib : true,
      opsiText: item?.opsi?.map((option) => option.label).join("; ") || "",
    });
    setDialog("question");
  }

  function saveQuestion(event) {
    event.preventDefault();

    return run(async () => {
      const form = questionForm;
      const order = Number(form.urutan);

      if (!form.teks_pertanyaan.trim()) {
        throw new Error("Teks pertanyaan wajib diisi.");
      }
      if (!Number.isInteger(order) || order < 1) {
        throw new Error("Urutan harus berupa angka bulat minimal 1.");
      }

      const options = optionsTypes.includes(form.tipe_jawaban)
        ? form.opsiText
            .split(";")
            .map((value) => value.trim())
            .filter(Boolean)
        : [];

      const payload = {
        jenis_kuesioner: jenis,
        teks_pertanyaan: form.teks_pertanyaan.trim(),
        tipe_jawaban: form.tipe_jawaban,
        urutan: order,
        wajib: form.wajib,
        opsi: options,
      };

      if (form.id) await api.put(`/pertanyaan/${form.id}`, payload);
      else await api.post("/pertanyaan", payload);

      setDialog(null);
      setSyncDirty((previous) => ({ ...previous, [jenis]: true }));
      setNotice(
        "Pertanyaan tersimpan. Kirim ke Google Form untuk menerapkannya.",
      );

      try {
        await refreshQuestions();
      } catch {
        setError("Pertanyaan tersimpan, tetapi daftar gagal dimuat ulang.");
      }
    });
  }

  function toggleQuestion(item) {
    return run(async () => {
      await api.patch(`/pertanyaan/${item.id}/aktif`, {
        is_aktif: !item.is_aktif,
      });

      setQuestions((previous) =>
        previous.map((q) =>
          q.id === item.id ? { ...q, is_aktif: !item.is_aktif } : q,
        ),
      );
      setSyncDirty((previous) => ({ ...previous, [jenis]: true }));
      setNotice(
        "Status pertanyaan tersimpan. Kirim ke Google Form untuk menerapkannya.",
      );
    });
  }

  function deleteQuestion(item) {
    if (!window.confirm(`Hapus pertanyaan "${item.teks_pertanyaan}"?`)) return;

    return run(async () => {
      await api.delete(`/pertanyaan/${item.id}`);

      setQuestions((previous) => previous.filter((q) => q.id !== item.id));
      setSyncDirty((previous) => ({ ...previous, [jenis]: true }));
      setNotice(
        "Pertanyaan dihapus. Kirim ke Google Form untuk menerapkannya.",
      );
    });
  }

  function chooseFile(event) {
    const selected = event.target.files?.[0] || null;

    if (!selected || lock.current) return;

    setPreview(null);
    setFile(null);
    setError("");
    setNotice("");

    if (!selected.name.toLowerCase().endsWith(".xlsx")) {
      setError("Gunakan file Excel dengan ekstensi .xlsx.");
      event.target.value = "";
      return;
    }

    if (selected.size > 5 * 1024 * 1024) {
      setError("Ukuran file maksimal 5 MB.");
      event.target.value = "";
      return;
    }

    setFile(selected);

    // File yang sama bisa dipilih kembali setelah diperbaiki.
    event.target.value = "";

    return run(async () => {
      const body = new FormData();
      body.append("file", selected);

      const response = await api.post("/pertanyaan/import/preview", body);

      setPreview(response.data.data);
    });
  }

  function previewImport() {
    return run(async () => {
      if (!file) return;

      const body = new FormData();
      body.append("file", file);

      const response = await api.post("/pertanyaan/import/preview", body);
      setPreview(response.data.data);
    });
  }

  function saveImport() {
    return run(async () => {
      if (!file || !preview?.bisa_disimpan) return;

      const body = new FormData();
      body.append("file", file);

      const response = await api.post("/pertanyaan/import", body);
      const data = response.data.data;

      setSyncDirty({ MINGGUAN: true, TAHUNAN: true });
      setPreview(null);
      setFile(null);
      setFileVersion((value) => value + 1);

      // TAMBAHAN: tutup popup setelah penyimpanan berhasil.
      setDialog(null);

      setNotice(
        `Import selesai: ${data.ditambah} baru, ${data.diperbarui} diperbarui, ${data.dinonaktifkan} dinonaktifkan. Kirim masing-masing jenis yang berubah ke Google Form.`,
      );

      try {
        await refreshQuestions();
      } catch {
        setError(
          "Import tersimpan, tetapi daftar pertanyaan gagal dimuat ulang.",
        );
      }
    });
  }

  function refreshSelected() {
    return run(async () => {
      await refreshQuestionnaires();
      setNotice("Status kuesioner berhasil dimuat ulang.");
    }, false);
  }

  const previewTitle = header.judul.replaceAll("{nama_kegiatan}", title);

  const replacements = {
    nama_kegiatan: title,
    tema: questionnaire?.tema || activity?.tema || "—",
    pembicara: questionnaire?.pembicara || activity?.pembicara || "—",
    tanggal: activity?.tanggal
      ? new Date(`${activity.tanggal}T12:00:00`).toLocaleDateString("id-ID")
      : "",
    jam_mulai: activity?.waktu_mulai || "",
    jam_selesai: activity?.waktu_selesai || "",
    komunitas: activity?.nama_komunitas || "",
    nats_alkitab: activity?.nats_alkitab || "",
    ditutup: displayTime(questionnaire?.ditutup_pada),
  };

  function renderTemplate(text) {
    return text.replace(/\{([^}]+)\}/g, (match, key) =>
      Object.hasOwn(replacements, key) ? replacements[key] : match,
    );
  }

  return {
    canEdit,
    jenis,
    changeJenis,
    tab,
    setTab,
    choices,
    activityID,
    setActivityID,
    activitySearch,
    setActivitySearch,
    year,
    setYear,
    title,
    activity,
    questionnaire,
    status,
    questions,
    header,
    headerDirty,
    updateHeader,
    renderTemplate,
    previewTitle,
    syncPending: Boolean(syncDirty[jenis]),
    loading,
    busy,
    error,
    notice,
    formLink,
    participantLink,
    dialog,
    setDialog,
    schedule,
    setSchedule,
    questionForm,
    setQuestionForm,
    file,
    preview,
    fileVersion,
    createWeekly,
    openSchedule,
    saveSchedule,
    toggleStatus,
    copyCode,
    prepareParticipantLink,
    refreshSelected,
    saveHeader,
    syncGoogle,
    openQuestion,
    saveQuestion,
    toggleQuestion,
    deleteQuestion,
    chooseFile,
    previewImport,
    saveImport,
  };
}
