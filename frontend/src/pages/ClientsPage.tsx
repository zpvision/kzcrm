import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Archive,
  ChevronRight,
  CircleDollarSign,
  History,
  MessageCircle,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Search,
  Users,
} from "lucide-react";
import { api } from "../api";
import type { Client, Status } from "../types";
import {
  cx,
  isoDate,
  Modal,
  MonthPicker,
  monthTitle,
  shiftMonth,
} from "../main";

const statusMeta: Record<Status, [string, string]> = {
  unpaid: ["Не оплачено", "bg-red-50 text-red-700 ring-red-100"],
  pending: [
    "Оплачено, ждёт продления",
    "bg-amber-50 text-amber-700 ring-amber-100",
  ],
  completed: [
    "Оплачено и продлено",
    "bg-emerald-50 text-emerald-700 ring-emerald-100",
  ],
};
function Badge({ status }: { status: Status }) {
  const [x, c] = statusMeta[status];
  return (
    <span
      className={cx(
        "inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ring-1 ring-inset",
        c,
      )}
    >
      {x}
    </span>
  );
}
type Props = {
  month: string;
  setMonth: (m: string) => void;
  notify: (s: string) => void;
};
export function ClientsPage({ month, setMonth, notify }: Props) {
  const [items, setItems] = useState<Client[]>([]);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<"all" | Status>("all");
  const [busy, setBusy] = useState(true);
  const [edit, setEdit] = useState<Client | true | null>(null);
  const [payment, setPayment] = useState<{
    client: Client;
    service?: string;
  } | null>(null);
  const [renewal, setRenewal] = useState<{
    client: Client;
    service: "finkoper" | "whatsapp";
  } | null>(null);
  const [drawer, setDrawer] = useState<Client | null>(null);
  const [history, setHistory] = useState<{
    client: Client;
    service: "finkoper" | "whatsapp";
  } | null>(null);
  const load = useCallback(async () => {
    setBusy(true);
    try {
      setItems(
        await api<Client[]>(
          `/clients?month=${month}&q=${encodeURIComponent(query)}`,
        ),
      );
    } catch (e) {
      notify((e as Error).message);
    } finally {
      setBusy(false);
    }
  }, [month, query, notify]);
  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);
  const shown = useMemo(
    () =>
      items.filter(
        (c) =>
          filter === "all" ||
          c.finKoperStatus === filter ||
          c.whatsAppStatus === filter,
      ),
    [items, filter],
  );
  const pending = items.filter(
    (x) => x.finKoperStatus === "pending" || x.whatsAppStatus === "pending",
  ).length;
  const paid = items.filter((x) => x.finKoperStatus !== "unpaid").length;
  const unpaid = items.filter((x) => x.finKoperStatus === "unpaid").length;
  const wa = items.filter((x) => x.whatsAppEnabled).length;
  return (
    <main className="p-4 md:p-6 xl:p-8">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold md:text-3xl">
            Клиенты и продления
          </h1>
          <p className="mt-1 text-slate-500">
            Подписки, оплаты и история операций
          </p>
        </div>
        <button className="btn-primary" onClick={() => setEdit(true)}>
          <Plus size={18} />
          Добавить нового клиента
        </button>
      </div>
      <div className="mb-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <Stat
          label="Всего клиентов"
          value={items.length}
          icon={<Users />}
          tone="blue"
        />
        <Stat
          label="Оплачено клиентов"
          value={paid}
          icon={<CircleDollarSign />}
          tone="green"
        />
        <Stat
          label="Не оплачено клиентов"
          value={unpaid}
          icon={<CircleDollarSign />}
          tone="red"
        />
        <Stat
          label="Ожидают продления"
          value={pending}
          icon={<RefreshCw />}
          tone="amber"
        />
        <Stat
          label="WhatsApp активен"
          value={wa}
          icon={<MessageCircle />}
          tone="green"
        />
      </div>
      <div className="card">
        <div className="flex flex-wrap items-center gap-3 border-b border-slate-100 p-4">
          <div className="relative min-w-60 flex-1">
            <Search
              className="absolute left-3 top-2.5 text-slate-400"
              size={18}
            />
            <input
              className="field pl-10"
              placeholder="Поиск по названию, ИИН/БИН, email, телефону"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
          <select
            className="field w-auto"
            value={filter}
            onChange={(e) => setFilter(e.target.value as typeof filter)}
          >
            <option value="all">Все статусы</option>
            <option value="unpaid">Не оплачено</option>
            <option value="pending">Ждёт продления</option>
            <option value="completed">Оплачено и продлено</option>
          </select>
          <MonthPicker month={month} onChange={setMonth} />
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[1150px]">
            <thead>
              <tr>
                <th className="th sticky left-0 z-20 min-w-60">Компания</th>
                <th className="th">Ответственный</th>
                <th className="th text-center">Клиентов</th>
                <th className="th border-l border-blue-200 bg-blue-100/80 text-blue-900">
                  FinKoper до
                </th>
                <th className="th border-r border-blue-200 bg-blue-100/80 text-blue-900">
                  Оплата FinKoper
                </th>
                <th className="th border-l border-emerald-200 bg-emerald-100/80 text-emerald-900">
                  WhatsApp
                </th>
                <th className="th bg-emerald-100 text-emerald-900">
                  WhatsApp до
                </th>
                <th className="th border-r border-emerald-200 bg-emerald-100/80 text-emerald-900">
                  Оплата WhatsApp
                </th>
                <th className="th sticky right-0 z-20">Действия</th>
              </tr>
            </thead>
            <tbody>
              {shown.map((c) => (
                <tr key={c.id} className="group hover:bg-slate-50/70">
                  <td className="td sticky left-0 z-10 bg-white group-hover:bg-slate-50">
                    <button
                      onClick={() => setDrawer(c)}
                      className="text-left hover:underline"
                    >
                      <span className="flex items-center gap-2 font-semibold text-brand-700">
                        {c.companyName}
                        <ChevronRight size={15} />
                      </span>
                      <span className="mt-1 block text-xs font-normal text-slate-400">
                        ИИН / БИН: {c.taxId || "—"}
                      </span>
                      <span className="mt-0.5 block text-xs font-normal text-slate-500">
                        {c.email}
                      </span>
                    </button>
                  </td>
                  <td className="td">{c.contactName || "—"}</td>
                  <td className="td text-center">
                    <span className="rounded-lg bg-slate-100 px-2.5 py-1 font-semibold">
                      {c.count ?? "—"}
                    </span>
                  </td>
                  <td className="td border-l border-blue-100 bg-blue-50/60 font-medium group-hover:bg-blue-100/60">
                    {isoDate(c.finKoperExpires)}
                  </td>
                  <td className="group/finpay td border-r border-blue-100 bg-blue-50/60 group-hover:bg-blue-100/60">
                    <div className="relative min-h-[52px]">
                      <Badge status={c.finKoperStatus} />
                      <button
                        type="button"
                        title="История оплат и продлений FinKoper"
                        aria-label="История оплат и продлений FinKoper"
                        className="absolute right-0 top-0 inline-flex h-7 w-7 items-center justify-center rounded-lg text-blue-600 hover:bg-blue-200/70"
                        onClick={() =>
                          setHistory({ client: c, service: "finkoper" })
                        }
                      >
                        <History size={15} />
                      </button>
                      <button
                        className="btn-secondary absolute left-0 top-7 z-10 !h-8 whitespace-nowrap !px-3 text-xs opacity-0 shadow-sm transition-opacity group-hover/finpay:opacity-100"
                        onClick={() =>
                          setRenewal({ client: c, service: "finkoper" })
                        }
                      >
                        <RefreshCw size={14} />
                        Продлить FinKoper
                      </button>
                    </div>
                  </td>
                  <td className="td border-l border-emerald-100 bg-emerald-50/60 group-hover:bg-emerald-100/60">
                    <span
                      className={cx(
                        "rounded-full px-2 py-1 text-xs font-semibold",
                        c.whatsAppEnabled
                          ? "bg-emerald-50 text-emerald-700"
                          : "bg-slate-100 text-slate-500",
                      )}
                    >
                      {c.whatsAppEnabled ? "Включён" : "Выключен"}
                    </span>
                  </td>
                  <td className="td bg-emerald-50 font-medium group-hover:bg-emerald-100">
                    {isoDate(c.whatsAppExpires)}
                  </td>
                  <td className="group/wapay td border-r border-emerald-100 bg-emerald-50/60 group-hover:bg-emerald-100/60">
                    <div className="relative min-h-[52px]">
                      <button
                        type="button"
                        title="История оплат и продлений WhatsApp"
                        aria-label="История оплат и продлений WhatsApp"
                        className="absolute right-0 top-0 inline-flex h-7 w-7 items-center justify-center rounded-lg text-emerald-700 hover:bg-emerald-200/70"
                        onClick={() =>
                          setHistory({ client: c, service: "whatsapp" })
                        }
                      >
                        <History size={15} />
                      </button>
                      {c.whatsAppEnabled ? (
                        <>
                          <Badge status={c.whatsAppStatus} />
                          <button
                            className="btn-secondary absolute left-0 top-7 z-10 !h-8 whitespace-nowrap !px-3 text-xs opacity-0 shadow-sm transition-opacity group-hover/wapay:opacity-100"
                            onClick={() =>
                              setRenewal({ client: c, service: "whatsapp" })
                            }
                          >
                            <RefreshCw size={14} />
                            Продлить WhatsApp
                          </button>
                        </>
                      ) : (
                        <span className="text-slate-400">—</span>
                      )}
                    </div>
                  </td>
                  <td className="td sticky right-0 bg-white group-hover:bg-slate-50">
                    <div className="flex items-center gap-2">
                      <button
                        title="Отметить оплату"
                        className="btn-secondary !h-9 whitespace-nowrap !px-3 text-xs"
                        onClick={() => setPayment({ client: c })}
                      >
                        <CircleDollarSign size={15} />
                        Отметить оплату
                      </button>
                      <button
                        title="Редактировать"
                        className="icon-btn"
                        onClick={() => setEdit(c)}
                      >
                        <MoreHorizontal size={18} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {busy && (
            <div className="p-10 text-center text-slate-400">Загрузка…</div>
          )}
          {!busy && !shown.length && (
            <div className="p-12 text-center">
              <Users className="mx-auto mb-3 text-slate-300" size={34} />
              <div className="font-semibold">Клиенты не найдены</div>
              <div className="mt-1 text-slate-500">
                Измените фильтр или добавьте первого клиента
              </div>
            </div>
          )}
        </div>
      </div>
      {edit && (
        <ClientModal
          client={edit === true ? undefined : edit}
          onClose={() => setEdit(null)}
          onSaved={() => {
            setEdit(null);
            load();
            notify("Данные клиента сохранены");
          }}
        />
      )}
      {payment && (
        <PaymentModal
          {...payment}
          month={month}
          onClose={() => setPayment(null)}
          onSaved={() => {
            setPayment(null);
            load();
            notify("Оплата зарегистрирована");
          }}
        />
      )}
      {renewal && (
        <RenewalModal
          {...renewal}
          onClose={() => setRenewal(null)}
          onSaved={() => {
            setRenewal(null);
            load();
            notify("Продление зафиксировано");
          }}
        />
      )}
      {drawer && (
        <ClientDrawer
          initial={drawer}
          onClose={() => setDrawer(null)}
          onRenew={(service) => setRenewal({ client: drawer, service })}
        />
      )}
      {history && (
        <PaymentHistoryModal {...history} onClose={() => setHistory(null)} />
      )}
    </main>
  );
}
function Stat({
  label,
  value,
  icon,
  tone,
}: {
  label: string;
  value: number;
  icon: React.ReactNode;
  tone: string;
}) {
  const c =
    tone === "blue"
      ? "bg-blue-50 text-blue-600"
      : tone === "amber"
        ? "bg-amber-50 text-amber-600"
        : tone === "red"
          ? "bg-red-50 text-red-600"
          : "bg-emerald-50 text-emerald-600";
  return (
    <div className="card flex items-center gap-4 p-5">
      <div
        className={cx(
          "flex h-11 w-11 items-center justify-center rounded-xl",
          c,
        )}
      >
        {icon}
      </div>
      <div>
        <div className="text-2xl font-bold">{value}</div>
        <div className="text-sm text-slate-500">{label}</div>
      </div>
    </div>
  );
}

const empty = {
  companyName: "",
  taxId: "",
  email: "",
  contactName: "",
  phone: "",
  comment: "",
  finKoperExpires: "",
  whatsAppEnabled: false,
  whatsAppEnabledAt: "",
  whatsAppExpires: "",
  version: 0,
};
function ClientModal({
  client,
  onClose,
  onSaved,
}: {
  client?: Client;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [f, setF] = useState({
    ...empty,
    ...client,
    finKoperExpires: client?.finKoperExpires?.substring(0, 10) || "",
    whatsAppEnabledAt: client?.whatsAppEnabledAt?.substring(0, 10) || "",
    whatsAppExpires: client?.whatsAppExpires?.substring(0, 10) || "",
  });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const change = (k: string, v: unknown) => setF((x) => ({ ...x, [k]: v }));
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(client ? `/clients/${client.id}` : "/clients", {
        method: client ? "PUT" : "POST",
        body: JSON.stringify({
          companyName: f.companyName,
          taxId: f.taxId,
          email: f.email,
          contactName: f.contactName,
          phone: f.phone,
          comment: f.comment,
          finKoperExpires: f.finKoperExpires || null,
          whatsAppEnabled: f.whatsAppEnabled,
          whatsAppEnabledAt: f.whatsAppEnabledAt || null,
          whatsAppExpires: f.whatsAppExpires || null,
          version: f.version,
        }),
      });
      onSaved();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function archive() {
    if (
      !client ||
      !confirm("Архивировать клиента? Финансовая история сохранится.")
    )
      return;
    try {
      await api(`/clients/${client.id}/archive`, { method: "POST" });
      onSaved();
    } catch (e) {
      setError((e as Error).message);
    }
  }
  return (
    <Modal
      title={client ? "Редактировать клиента" : "Новый клиент"}
      onClose={onClose}
      wide
    >
      <form onSubmit={save}>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Наименование компании *"
            value={f.companyName}
            set={(v) => change("companyName", v)}
          />
          <Field
            label="ИИН / БИН"
            value={f.taxId || ""}
            set={(v) => change("taxId", v)}
          />
          <Field
            label="Почта / логин *"
            type="email"
            value={f.email}
            set={(v) => change("email", v)}
          />
          <Field
            label="Имя ответственного"
            value={f.contactName || ""}
            set={(v) => change("contactName", v)}
          />
          <Field
            label="Телефон"
            value={f.phone || ""}
            set={(v) => change("phone", v)}
          />
          <Field
            label="FinKoper до"
            type="date"
            value={f.finKoperExpires || ""}
            set={(v) => change("finKoperExpires", v)}
          />
        </div>
        <label className="mt-4 flex items-center gap-2 font-medium">
          <input
            type="checkbox"
            checked={f.whatsAppEnabled}
            onChange={(e) => change("whatsAppEnabled", e.target.checked)}
            className="h-4 w-4"
          />
          Интеграция WhatsApp включена
        </label>
        {f.whatsAppEnabled && (
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <Field
              label="Дата включения WhatsApp *"
              type="date"
              value={f.whatsAppEnabledAt || ""}
              set={(v) => change("whatsAppEnabledAt", v)}
            />
            <Field
              label="WhatsApp до"
              type="date"
              value={f.whatsAppExpires || ""}
              set={(v) => change("whatsAppExpires", v)}
            />
          </div>
        )}
        <label className="label mt-4">Комментарий</label>
        <textarea
          className="field h-24 py-2"
          value={f.comment || ""}
          onChange={(e) => change("comment", e.target.value)}
        />
        {error && <div className="mt-3 text-sm text-red-600">{error}</div>}
        <div className="mt-5 flex justify-between">
          <div>
            {client && (
              <button
                type="button"
                className="btn text-red-600 hover:bg-red-50"
                onClick={archive}
              >
                <Archive size={17} />
                Архивировать
              </button>
            )}
          </div>
          <div className="flex gap-2">
            <button type="button" className="btn-secondary" onClick={onClose}>
              Отмена
            </button>
            <button disabled={busy} className="btn-primary">
              {busy ? "Сохранение…" : "Сохранить"}
            </button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
function Field({
  label,
  value,
  set,
  type = "text",
}: {
  label: string;
  value: string;
  set: (v: string) => void;
  type?: string;
}) {
  return (
    <label>
      <span className="label">{label}</span>
      <input
        required={label.includes("*")}
        type={type}
        className="field"
        value={value}
        onChange={(e) => set(e.target.value)}
      />
    </label>
  );
}

function PaymentModal({
  client,
  service: initial,
  month,
  onClose,
  onSaved,
}: {
  client: Client;
  service?: string;
  month: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [service, setService] = useState(initial || "finkoper");
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [billing, setBilling] = useState(month);
  const billingMonths = useMemo(
    () =>
      Array.from({ length: 85 }, (_, index) => {
        const value = shiftMonth(month, index - 24);
        return { value, label: monthTitle(value) };
      }),
    [month],
  );
  const [months, setMonths] = useState(1);
  const [amount, setAmount] = useState("");
  const [comment, setComment] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api("/payments", {
        method: "POST",
        body: JSON.stringify({
          clientId: client.id,
          service,
          receivedOn: date,
          billingMonth: billing,
          months,
          amount: amount ? Number(amount) : null,
          comment,
          idempotencyKey: crypto.randomUUID(),
        }),
      });
      onSaved();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title="Отметить оплату" onClose={onClose}>
      <div className="mb-4 rounded-xl bg-slate-50 p-3">
        <div className="text-xs text-slate-500">Клиент</div>
        <div className="font-semibold">{client.companyName}</div>
      </div>
      <form onSubmit={submit} className="space-y-4">
        <div>
          <span className="label">Услуга</span>
          <div
            className={cx(
              "grid gap-2 rounded-xl bg-slate-100 p-1",
              client.whatsAppEnabled ? "grid-cols-2" : "grid-cols-1",
            )}
          >
            <button
              type="button"
              onClick={() => setService("finkoper")}
              className={cx(
                "h-10 rounded-lg font-semibold transition",
                service === "finkoper"
                  ? "bg-white text-brand-700 shadow-sm ring-1 ring-brand-200"
                  : "text-slate-500 hover:text-slate-800",
              )}
            >
              FinKoper
            </button>
            {client.whatsAppEnabled && (
              <button
                type="button"
                onClick={() => setService("whatsapp")}
                className={cx(
                  "h-10 rounded-lg font-semibold transition",
                  service === "whatsapp"
                    ? "bg-white text-emerald-700 shadow-sm ring-1 ring-emerald-200"
                    : "text-slate-500 hover:text-slate-800",
                )}
              >
                WhatsApp
              </button>
            )}
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Дата оплаты *" type="date" value={date} set={setDate} />
          <label>
            <span className="label">Расчётный месяц</span>
            <select
              className="field"
              value={billing}
              onChange={(e) => setBilling(e.target.value)}
              required
            >
              {billingMonths.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div>
          <span className="label">Оплачено месяцев</span>
          <div className="flex gap-2">
            {[1, 3, 6, 12].map((n) => (
              <button
                key={n}
                type="button"
                onClick={() => setMonths(n)}
                className={cx(
                  "h-9 flex-1 rounded-lg border font-semibold",
                  months === n
                    ? "border-brand-500 bg-brand-50 text-brand-700"
                    : "border-slate-200",
                )}
              >
                {n}
              </button>
            ))}
            <input
              className="field !h-9 w-20"
              type="number"
              min="1"
              max="120"
              value={months}
              onChange={(e) => setMonths(Number(e.target.value))}
            />
          </div>
        </div>
        <Field
          label="Сумма, KZT (необязательно)"
          type="number"
          value={amount}
          set={setAmount}
        />
        <label>
          <span className="label">Комментарий</span>
          <textarea
            className="field h-20 py-2"
            value={comment}
            onChange={(e) => setComment(e.target.value)}
          />
        </label>
        <div className="rounded-xl bg-blue-50 p-3 text-xs text-blue-800">
          Оплата покроет {months} расчётных мес. начиная с {billing}. Дата
          подписки не изменится.
        </div>
        {error && <div className="text-sm text-red-600">{error}</div>}
        <div className="flex justify-end gap-2">
          <button type="button" className="btn-secondary" onClick={onClose}>
            Отмена
          </button>
          <button className="btn-primary" disabled={busy}>
            {busy ? "Сохранение…" : "Подтвердить оплату"}
          </button>
        </div>
      </form>
    </Modal>
  );
}

type EventItem = {
  kind: "payment" | "renewal";
  id: string;
  service: string;
  date: string;
  period: string;
  months: number;
  amount?: number;
  currency?: string;
  state: string;
  user: string;
  comment?: string;
  paymentId?: string;
};
const auditActionLabels: Record<string, string> = {
  "client.created": "Клиент создан",
  "client.updated": "Данные клиента изменены",
  "client.archived": "Клиент архивирован",
  "payment.created": "Оплата добавлена",
  "payment.cancelled": "Оплата отменена",
  "renewal.created": "Продление зафиксировано",
  "renewal.cancelled": "Продление отменено",
  "count.updated": "Количество клиентов изменено",
  "count.corrected": "Количество клиентов скорректировано",
  "month.confirmed": "Количество за месяц подтверждено",
};
function auditActionLabel(value: unknown) {
  const action = String(value);
  return auditActionLabels[action] || action;
}
function addMonths(value: string, months: number) {
  const d = new Date(value + "T12:00:00");
  const day = d.getDate();
  d.setDate(1);
  d.setMonth(d.getMonth() + months);
  const last = new Date(d.getFullYear(), d.getMonth() + 1, 0).getDate();
  d.setDate(Math.min(day, last));
  return d.toISOString().slice(0, 10);
}
function RenewalModal({
  client,
  service,
  onClose,
  onSaved,
}: {
  client: Client;
  service: "finkoper" | "whatsapp";
  onClose: () => void;
  onSaved: () => void;
}) {
  const current =
    (service === "finkoper"
      ? client.finKoperExpires
      : client.whatsAppExpires
    )?.slice(0, 10) || new Date().toISOString().slice(0, 10);
  const [months, setMonths] = useState(1);
  const [newDate, setNewDate] = useState(addMonths(current, 1));
  const [paymentId, setPaymentId] = useState("");
  const [payments, setPayments] = useState<EventItem[]>([]);
  const [comment, setComment] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    api<EventItem[]>(`/clients/${client.id}/timeline`).then((x) =>
      setPayments(
        x.filter(
          (e) =>
            e.kind === "payment" &&
            e.service === service &&
            e.state === "active" &&
            !x.some(
              (r) =>
                r.kind === "renewal" &&
                r.state === "active" &&
                r.paymentId === e.id,
            ),
        ),
      ),
    );
  }, [client.id, service]);
  function choose(n: number) {
    setMonths(n);
    setNewDate(addMonths(current, n));
    setPaymentId(payments.find((payment) => payment.months === n)?.id || "");
  }
  function choosePayment(id: string) {
    setPaymentId(id);
    const p = payments.find((x) => x.id === id);
    if (p) {
      setMonths(p.months);
      setNewDate(addMonths(current, p.months));
    }
  }
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (
      !paymentId &&
      !confirm("Связанная оплата не выбрана. Зафиксировать ручное продление?")
    )
      return;
    setBusy(true);
    setError("");
    try {
      await api("/renewals", {
        method: "POST",
        body: JSON.stringify({
          clientId: client.id,
          service,
          months,
          newExpires: newDate,
          paymentId: paymentId || null,
          comment,
          idempotencyKey: crypto.randomUUID(),
        }),
      });
      onSaved();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title={`Продлить ${service === "finkoper" ? "FinKoper" : "WhatsApp"}`}
      onClose={onClose}
    >
      <div className="mb-4 rounded-xl bg-slate-50 p-3">
        <div className="font-semibold">{client.companyName}</div>
        <div className="mt-1 text-sm text-slate-500">
          Текущая дата: <b className="text-slate-700">{isoDate(current)}</b>
        </div>
      </div>
      <form onSubmit={submit} className="space-y-4">
        <div>
          <span className="label">Срок продления</span>
          <div className="grid grid-cols-4 gap-2">
            {[1, 3, 6, 12].map((n) => (
              <button
                key={n}
                type="button"
                onClick={() => choose(n)}
                className={cx(
                  "h-10 rounded-lg border font-semibold",
                  months === n
                    ? "border-brand-500 bg-brand-50 text-brand-700"
                    : "border-slate-200",
                )}
              >
                {n} мес.
              </button>
            ))}
          </div>
        </div>
        <Field
          label="Новая дата окончания *"
          type="date"
          value={newDate}
          set={setNewDate}
        />
        <label>
          <span className="label">Связанная оплата</span>
          <select
            className="field"
            value={paymentId}
            onChange={(e) => choosePayment(e.target.value)}
          >
            <option value="">Без связанной оплаты</option>
            {payments.map((p) => (
              <option key={p.id} value={p.id}>
                {new Date(p.date).toLocaleDateString("ru-RU")} · {p.period}
                {p.amount ? ` · ${p.amount} KZT` : ""}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span className="label">Комментарий</span>
          <textarea
            className="field h-20 py-2"
            value={comment}
            onChange={(e) => setComment(e.target.value)}
          />
        </label>
        <div className="rounded-xl bg-emerald-50 p-3 text-sm text-emerald-800">
          После сохранения дата изменится на <b>{isoDate(newDate)}</b>.
        </div>
        {error && <div className="text-sm text-red-600">{error}</div>}
        <div className="flex justify-end gap-2">
          <button type="button" className="btn-secondary" onClick={onClose}>
            Отмена
          </button>
          <button className="btn-primary" disabled={busy}>
            {busy ? "Сохранение…" : "Зафиксировать продление"}
          </button>
        </div>
      </form>
    </Modal>
  );
}

function ClientDrawer({
  initial,
  onClose,
  onRenew,
}: {
  initial: Client;
  onClose: () => void;
  onRenew: (s: "finkoper" | "whatsapp") => void;
}) {
  const [tab, setTab] = useState<"info" | "timeline" | "audit">("info");
  const [events, setEvents] = useState<EventItem[]>([]);
  const [audit, setAudit] = useState<Record<string, unknown>[]>([]);
  useEffect(() => {
    if (tab === "timeline")
      api<EventItem[]>(`/clients/${initial.id}/timeline`).then(setEvents);
    if (tab === "audit")
      api<Record<string, unknown>[]>(`/clients/${initial.id}/audit`).then(
        setAudit,
      );
  }, [tab, initial.id]);
  return (
    <>
      <div className="fixed inset-0 z-40 bg-slate-950/20" onClick={onClose} />
      <aside className="fixed inset-y-0 right-0 z-50 w-full max-w-xl overflow-y-auto border-l border-slate-200 bg-white shadow-2xl">
        <div className="sticky top-0 z-10 border-b border-slate-200 bg-white px-6 pb-0 pt-5">
          <div className="flex items-start justify-between">
            <div>
              <div className="text-xs font-semibold uppercase tracking-wider text-brand-600">
                Карточка клиента
              </div>
              <h2 className="mt-1 text-2xl font-bold">{initial.companyName}</h2>
              <div className="mt-1 text-sm text-slate-500">{initial.email}</div>
            </div>
            <button className="icon-btn" onClick={onClose}>
              ×
            </button>
          </div>
          <div className="mt-5 flex gap-5 text-sm">
            {[
              ["info", "Общая информация"],
              ["timeline", "Оплаты и продления"],
              ["audit", "История действий"],
            ].map(([k, v]) => (
              <button
                key={k}
                onClick={() => setTab(k as typeof tab)}
                className={cx(
                  "border-b-2 pb-3 font-semibold",
                  tab === k
                    ? "border-brand-600 text-brand-700"
                    : "border-transparent text-slate-500",
                )}
              >
                {v}
              </button>
            ))}
          </div>
        </div>
        <div className="p-6">
          {tab === "info" && (
            <div className="space-y-5">
              <div className="card grid grid-cols-2 gap-4 p-4">
                <Info label="ИИН / БИН" value={initial.taxId} />
                <Info label="Ответственный" value={initial.contactName} />
                <Info label="Телефон" value={initial.phone} />
                <Info label="Комментарий" value={initial.comment} />
              </div>
              <ServiceCard
                title="FinKoper"
                expires={initial.finKoperExpires}
                status={initial.finKoperStatus}
                onRenew={() => onRenew("finkoper")}
              />
              <ServiceCard
                title="WhatsApp"
                enabledAt={initial.whatsAppEnabledAt}
                expires={initial.whatsAppExpires}
                status={initial.whatsAppStatus}
                disabled={!initial.whatsAppEnabled}
                onRenew={() => onRenew("whatsapp")}
              />
            </div>
          )}
          {tab === "timeline" && <Timeline items={events} />}{" "}
          {tab === "audit" && (
            <div className="space-y-3">
              {audit.map((a, i) => (
                <div key={i} className="rounded-xl border border-slate-200 p-4">
                  <div className="flex justify-between gap-3">
                    <b>{auditActionLabel(a.action)}</b>
                    <span className="text-xs text-slate-400">
                      {new Date(String(a.date)).toLocaleString("ru-RU")}
                    </span>
                  </div>
                  <div className="mt-1 text-sm text-slate-500">
                    {String(a.user)}
                  </div>
                  <details className="mt-2 text-xs text-slate-500">
                    <summary className="cursor-pointer">Изменения</summary>
                    <pre className="mt-2 overflow-auto whitespace-pre-wrap">
                      {JSON.stringify({ old: a.old, new: a.new }, null, 2)}
                    </pre>
                  </details>
                </div>
              ))}
            </div>
          )}
        </div>
      </aside>
    </>
  );
}
function Info({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <div className="text-xs text-slate-400">{label}</div>
      <div className="mt-1 font-medium">{value || "—"}</div>
    </div>
  );
}
function ServiceCard({
  title,
  enabledAt,
  expires,
  status,
  disabled,
  onRenew,
}: {
  title: string;
  enabledAt?: string;
  expires?: string;
  status: Status;
  disabled?: boolean;
  onRenew: () => void;
}) {
  return (
    <div className="card p-5">
      <div className="flex items-start justify-between">
        <div>
          <h3 className="font-bold">{title}</h3>
          {enabledAt && (
            <div className="mt-2 text-sm text-slate-500">
              Включён: <b className="text-slate-700">{isoDate(enabledAt)}</b>
            </div>
          )}
          <div className="mt-2 text-sm text-slate-500">Действует до</div>
          <div className="text-lg font-bold">
            {disabled ? "Интеграция выключена" : isoDate(expires)}
          </div>
        </div>
        {!disabled && <Badge status={status} />}
      </div>
      {!disabled && (
        <button className="btn-secondary mt-4 w-full" onClick={onRenew}>
          <RefreshCw size={16} />
          Продлить {title}
        </button>
      )}
    </div>
  );
}
function Timeline({ items }: { items: EventItem[] }) {
  return (
    <div className="relative space-y-3 before:absolute before:bottom-4 before:left-[17px] before:top-4 before:w-px before:bg-slate-200">
      {items.map((e) => (
        <div key={e.id} className="relative flex gap-4">
          <div
            className={cx(
              "z-10 mt-3 flex h-9 w-9 shrink-0 items-center justify-center rounded-full",
              e.kind === "payment"
                ? "bg-blue-100 text-blue-600"
                : "bg-emerald-100 text-emerald-600",
            )}
          >
            {e.kind === "payment" ? (
              <CircleDollarSign size={17} />
            ) : (
              <RefreshCw size={17} />
            )}
          </div>
          <div className="flex-1 rounded-xl border border-slate-200 p-4">
            <div className="flex justify-between gap-3">
              <b>
                {e.kind === "payment" ? "Оплата" : "Продление"} ·{" "}
                {e.service === "finkoper" ? "FinKoper" : "WhatsApp"}
              </b>
              <span className="text-xs text-slate-400">
                {new Date(e.date).toLocaleString("ru-RU")}
              </span>
            </div>
            <div className="mt-1 text-sm text-slate-600">
              {e.period}
              {e.amount
                ? ` · ${e.amount.toLocaleString("ru-RU")} ${e.currency}`
                : ""}
            </div>
            <div className="mt-2 text-xs text-slate-400">
              {e.user}
              {e.state === "cancelled" ? " · Отменено" : ""}
            </div>
          </div>
        </div>
      ))}
      {!items.length && (
        <div className="py-12 text-center text-slate-400">Событий пока нет</div>
      )}
    </div>
  );
}

function PaymentHistoryModal({
  client,
  service,
  onClose,
}: {
  client: Client;
  service: "finkoper" | "whatsapp";
  onClose: () => void;
}) {
  const [items, setItems] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  useEffect(() => {
    setLoading(true);
    api<EventItem[]>(`/clients/${client.id}/timeline`)
      .then((events) =>
        setItems(events.filter((event) => event.service === service)),
      )
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [client.id, service]);
  return (
    <Modal
      title={`История ${service === "finkoper" ? "FinKoper" : "WhatsApp"}`}
      onClose={onClose}
      wide
    >
      <div className="mb-4 rounded-xl bg-slate-50 p-3">
        <div className="font-semibold">{client.companyName}</div>
        <div className="mt-1 text-xs text-slate-500">Оплаты и продления</div>
      </div>
      {loading ? (
        <div className="py-12 text-center text-slate-400">Загрузка…</div>
      ) : error ? (
        <div className="rounded-xl bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-200">
          <table className="w-full min-w-[760px]">
            <thead>
              <tr>
                <th className="th">Дата</th>
                <th className="th">Операция</th>
                <th className="th">Период</th>
                <th className="th">Сумма</th>
                <th className="th">Сотрудник</th>
                <th className="th">Состояние</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr
                  key={`${item.kind}-${item.id}`}
                  className="hover:bg-slate-50"
                >
                  <td className="td text-sm">
                    {new Date(item.date).toLocaleString("ru-RU")}
                  </td>
                  <td className="td font-semibold">
                    {item.kind === "payment" ? "Оплата" : "Продление"}
                  </td>
                  <td className="td">
                    <div>{item.period}</div>
                    {item.comment && (
                      <div className="mt-1 max-w-64 truncate text-xs text-slate-400">
                        {item.comment}
                      </div>
                    )}
                  </td>
                  <td className="td font-medium">
                    {item.amount != null
                      ? `${item.amount.toLocaleString("ru-RU")} ${item.currency || "KZT"}`
                      : "—"}
                  </td>
                  <td className="td text-slate-600">{item.user}</td>
                  <td className="td">
                    <span
                      className={cx(
                        "rounded-full px-2.5 py-1 text-xs font-semibold",
                        item.state === "cancelled"
                          ? "bg-red-50 text-red-700"
                          : "bg-emerald-50 text-emerald-700",
                      )}
                    >
                      {item.state === "cancelled" ? "Отменено" : "Активно"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!items.length && (
            <div className="py-12 text-center text-slate-400">
              Операций пока нет
            </div>
          )}
        </div>
      )}
    </Modal>
  );
}
