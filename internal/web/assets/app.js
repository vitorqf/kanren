// kanren board client. Fetches columns + cards, renders draggable lists, and
// persists a drag as a POST to the same store the CLI writes (WEB-02).
"use strict";

const accentVar = { doing: "--doing", done: "--done" };

async function load() {
  const [cols, cards] = await Promise.all([
    fetch("/api/columns").then((r) => r.json()),
    fetch("/api/cards").then((r) => r.json()),
  ]);
  render(cols, cards || []);
}

function render(columns, cards) {
  const board = document.getElementById("board");
  board.replaceChildren();

  const byCol = new Map(columns.map((c) => [c, []]));
  for (const card of cards) {
    if (byCol.has(card.status)) byCol.get(card.status).push(card);
  }
  document.getElementById("summary").textContent =
    `${cards.length} card${cards.length === 1 ? "" : "s"} · ${columns.length} columns`;

  for (const col of columns) {
    const items = byCol.get(col);
    const accent = getComputedStyle(document.documentElement)
      .getPropertyValue(accentVar[col] || "--todo");

    const column = el("section", "column");
    column.style.setProperty("--accent", accent.trim());

    const head = el("div", "head");
    head.append(el("span", "dot"), textEl("span", "name", col),
      textEl("span", "count", String(items.length)));
    column.append(head);

    const list = el("div", "cards");
    list.dataset.status = col;
    if (items.length === 0) {
      list.append(textEl("div", "empty", "nothing here"));
    } else {
      for (const card of items) list.append(cardEl(card));
    }
    column.append(list);
    board.append(column);

    makeSortable(list);
  }
}

function cardEl(card) {
  const node = el("article", "card");
  node.dataset.id = card.id;
  node.append(textEl("div", "title", card.title));

  const meta = el("div", "meta");
  meta.append(textEl("span", "id", `#${card.id}`));
  for (const tag of card.tags || []) meta.append(textEl("span", "tag", tag));
  if (card.assignee) meta.append(textEl("span", "who", `@${card.assignee}`));
  node.append(meta);
  return node;
}

function makeSortable(list) {
  new Sortable(list, {
    group: "board",
    animation: 160,
    easing: "cubic-bezier(0.22, 1, 0.36, 1)",
    ghostClass: "sortable-ghost",
    chosenClass: "sortable-chosen",
    dragClass: "sortable-drag",
    onAdd: (evt) => {
      list.classList.remove("drop-active");
      move(evt.item.dataset.id, list.dataset.status);
    },
    onRemove: () => refreshCounts(),
  });
}

async function move(id, status) {
  try {
    const resp = await fetch(`/api/cards/${id}/move`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status }),
    });
    if (!resp.ok) throw new Error(await resp.text());
    toast(`#${id} → ${status}`);
    refreshCounts();
  } catch (err) {
    toast(String(err).slice(0, 80), true);
    load(); // reconcile UI with the source of truth on failure
  }
}

function refreshCounts() {
  for (const list of document.querySelectorAll(".cards")) {
    const n = list.querySelectorAll(".card").length;
    const count = list.parentElement.querySelector(".count");
    if (count) count.textContent = String(n);
    const empty = list.querySelector(".empty");
    if (n === 0 && !empty) list.append(textEl("div", "empty", "nothing here"));
    if (n > 0 && empty) empty.remove();
  }
}

// --- tiny DOM helpers ---
function el(tag, cls) {
  const n = document.createElement(tag);
  if (cls) n.className = cls;
  return n;
}
function textEl(tag, cls, text) {
  const n = el(tag, cls);
  n.textContent = text;
  return n;
}

let toastTimer;
function toast(msg, isErr) {
  const t = document.getElementById("toast");
  t.textContent = msg;
  t.classList.toggle("err", !!isErr);
  t.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => t.classList.remove("show"), 2200);
}

// Theme toggle: system default, overridable, remembered.
function initTheme() {
  const saved = localStorage.getItem("kanren-theme");
  if (saved) document.documentElement.dataset.theme = saved;
  document.getElementById("theme").addEventListener("click", () => {
    const cur = document.documentElement.dataset.theme
      || (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
    const next = cur === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = next;
    localStorage.setItem("kanren-theme", next);
  });
}

// Live reload: the server streams a "reload" event when any card file changes
// on disk (CLI edit, git pull, or another board's move).
function initLiveReload() {
  const es = new EventSource("/events");
  es.onmessage = () => load();
}

initTheme();
initLiveReload();
load();
