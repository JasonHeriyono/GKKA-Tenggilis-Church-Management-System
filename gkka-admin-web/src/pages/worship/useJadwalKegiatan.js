import { useEffect, useRef, useState } from 'react';
import api from '../../services/api';
import { getPetugasSummary } from '../../services/petugasSummary';

export function localDate(date) {
  const pad = number => String(number).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

export function emptyActivity(item = null) {
  return {
    nama_kegiatan: item?.nama_kegiatan || '',
    tanggal: item?.tanggal || localDate(new Date()),
    waktu_mulai: item?.waktu_mulai?.slice(0, 5) || '',
    waktu_selesai: item?.waktu_selesai?.slice(0, 5) || '',
    tema: item?.tema || '',
    nats_alkitab: item?.nats_alkitab || '',
    pembicara: item?.pembicara || '',
    lokasi: item?.lokasi || '',
    deskripsi: item?.deskripsi || '',
    id_komunitas: item?.id_komunitas
      ? String(item.id_komunitas)
      : '',
    perlu_evaluasi: Boolean(item?.perlu_evaluasi),
  };
}

function errorMessage(error) {
  return error.response?.data?.message || 'Tidak dapat terhubung ke server.';
}

async function allPages(url, params, signal) {
  const items = [];
  let page = 1;

  while (true) {
    const response = await api.get(url, {
      params: { ...params, page, limit: 100 },
      signal,
    });

    if (signal?.aborted) return [];

    const data = response.data.data;
    items.push(...(data.items || []));

    if (page >= (data.meta?.total_pages || 1)) return items;
    page += 1;
  }
}

export default function useJadwalKegiatan() {
  const [month, setMonth] = useState(
    () => new Date(new Date().getFullYear(), new Date().getMonth(), 1),
  );
  const [items, setItems] = useState([]);
  const [communities, setCommunities] = useState([]);
  const [communityFilter, setCommunityFilter] = useState('');
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState({});
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [revision, setRevision] = useState(0);

  const [detail, setDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const detailVersion = useRef(0);

  const [modal, setModal] = useState(null);
  const [form, setForm] = useState(emptyActivity);
  const [expanded, setExpanded] = useState(false);
  const [formError, setFormError] = useState('');
  const [fieldErrors, setFieldErrors] = useState({});

  const [busy, setBusy] = useState(false);
  const lock = useRef(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const year = month.getFullYear();
  const monthIndex = month.getMonth();
  const start = localDate(new Date(year, monthIndex, 1));
  const end = localDate(new Date(year, monthIndex + 1, 0));

  useEffect(() => {
    const controller = new AbortController();

    allPages('/komunitas', {}, controller.signal)
      .then(result => {
        if (!controller.signal.aborted) setCommunities(result);
      })
      .catch(err => {
        if (!controller.signal.aborted) {
          setError(`Daftar komunitas gagal dimuat. ${errorMessage(err)}`);
        }
      });

    return () => controller.abort();
  }, [revision]);

  useEffect(() => {
    const controller = new AbortController();

    async function load() {
      setLoading(true);
      setError('');
      setSummary({});
      setSummaryLoading(false);

      try {
        const result = await allPages(
          '/kegiatan',
          { dari: start, sampai: end },
          controller.signal,
        );

        if (controller.signal.aborted) return;
        setItems(result);
        setLoading(false);
        setSummaryLoading(true);

        try {
          const counts = await getPetugasSummary(result, controller.signal);

          if (!controller.signal.aborted) setSummary(counts);
        } catch {
          if (!controller.signal.aborted) {
            setError(
              'Daftar kegiatan berhasil dimuat, tetapi jumlah petugas belum tersedia. Tekan Muat ulang untuk mencoba kembali.',
            );
          }
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          setItems([]);
          setError(errorMessage(err));
        }
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
          setSummaryLoading(false);
        }
      }
    }

    load();
    return () => controller.abort();
  }, [start, end, revision]);

  const visibleItems = items.filter(item => {
    const matchesCommunity =
      communityFilter === '' ||
      (communityFilter === 'umum'
        ? !item.id_komunitas
        : String(item.id_komunitas) === communityFilter);

    const text = [
      item.nama_kegiatan,
      item.tema,
      item.pembicara,
      item.lokasi,
      item.nama_komunitas,
    ].filter(Boolean).join(' ').toLowerCase();

    return matchesCommunity && text.includes(search.trim().toLowerCase());
  });

  const groups = Object.entries(
    visibleItems.reduce((result, item) => {
      (result[item.tanggal] ||= []).push(item);
      return result;
    }, {}),
  ).sort(([a], [b]) => a.localeCompare(b));

  function closeDetail() {
    detailVersion.current += 1;
    setDetail(null);
    setDetailLoading(false);
  }

  function moveMonth(offset) {
    if (lock.current) return;
    closeDetail();
    setMonth(previous =>
      new Date(previous.getFullYear(), previous.getMonth() + offset, 1),
    );
  }

  function thisMonth() {
    if (lock.current) return;
    closeDetail();
    const now = new Date();
    setMonth(new Date(now.getFullYear(), now.getMonth(), 1));
  }

  async function openDetail(item) {
    if (lock.current) return;

    const version = ++detailVersion.current;
    setDetail(null);
    setDetailLoading(true);

    try {
      const response = await api.get(`/kegiatan/${item.id}`);

      if (version === detailVersion.current) {
        setDetail(response.data.data);
      }
    } catch (err) {
      if (version === detailVersion.current) setError(errorMessage(err));
    } finally {
      if (version === detailVersion.current) setDetailLoading(false);
    }
  }

  function openForm(mode) {
    if (lock.current || (mode === 'edit' && !detail)) return;
    setForm(emptyActivity(mode === 'edit' ? detail : null));
    setExpanded(mode === 'edit');
    setFormError('');
    setFieldErrors({});
    setModal(mode);
  }

  function changeField(key, value) {
    setForm(previous => ({ ...previous, [key]: value }));
  }

  function closeModal() {
    if (!lock.current) setModal(null);
  }

  async function save(event) {
    event.preventDefault();
    if (lock.current) return;

    const errors = {};

    if (!form.nama_kegiatan.trim()) {
      errors.nama_kegiatan = 'Nama kegiatan wajib diisi.';
    }
    if (!form.tanggal) errors.tanggal = 'Tanggal wajib diisi.';
    if (!form.waktu_mulai) errors.waktu_mulai = 'Jam mulai wajib diisi.';
    if (form.waktu_selesai && form.waktu_selesai <= form.waktu_mulai) {
      errors.waktu_selesai = 'Jam selesai harus setelah jam mulai.';
      setExpanded(true);
    }

    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) return;

    lock.current = true;
    setBusy(true);
    setFormError('');
    setNotice('');

    try {
      const payload = {
        ...form,
        nama_kegiatan: form.nama_kegiatan.trim(),
        id_komunitas: form.id_komunitas ? Number(form.id_komunitas) : null,
      };

      const response = modal === 'edit'
        ? await api.put(`/kegiatan/${detail.id}`, payload)
        : await api.post('/kegiatan', payload);

      const saved = response.data.data;
      const savedDate = new Date(`${saved.tanggal}T12:00:00`);

      detailVersion.current += 1;
      setDetailLoading(false);
      setDetail(saved);
      setMonth(new Date(savedDate.getFullYear(), savedDate.getMonth(), 1));
      setModal(null);
      setNotice('Kegiatan berhasil disimpan.');
      setRevision(value => value + 1);
    } catch (err) {
      setFormError(errorMessage(err));
      setFieldErrors(err.response?.data?.errors || {});
      setExpanded(true);
    } finally {
      lock.current = false;
      setBusy(false);
    }
  }

  async function remove() {
    if (!detail || lock.current) return;
    if (!window.confirm(`Hapus kegiatan "${detail.nama_kegiatan}"?`)) return;

    lock.current = true;
    setBusy(true);
    setError('');
    setNotice('');

    try {
      await api.delete(`/kegiatan/${detail.id}`);
      closeDetail();
      setNotice('Kegiatan berhasil dihapus.');
      setRevision(value => value + 1);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      lock.current = false;
      setBusy(false);
    }
  }

  return {
    month, moveMonth, thisMonth,
    groups, visibleItems, items,
    communities, communityFilter, setCommunityFilter, search, setSearch,
    loading, summary, summaryLoading,
    detail, detailLoading, openDetail, closeDetail,
    modal, openForm, closeModal, form, changeField,
    expanded, setExpanded, formError, fieldErrors, save,
    busy, error, notice, remove,
    reload: () => setRevision(value => value + 1),
  };
}