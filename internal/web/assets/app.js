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
    column.append(list, addRow(col));
    board.append(column);

    makeSortable(list);
  }
}

// addRow builds the inline "+ add card" affordance for a column.
function addRow(status) {
  const form = el("form", "add");
  form.dataset.status = status;
  const input = el("input", "add-input");
  input.type = "text";
  input.placeholder = "+ add card";
  input.setAttribute("aria-label", `add card to ${status}`);
  form.append(input);
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const title = input.value.trim();
    if (!title) return;
    input.value = "";
    await create(title, status);
  });
  return form;
}

async function create(title, status) {
  try {
    const resp = await fetch("/api/cards", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title, status }),
    });
    if (!resp.ok) throw new Error(await resp.text());
    const card = await resp.json();
    toast(`added #${card.id}`);
    await load(); // SSE will also fire; explicit load keeps it snappy
    focusAdd(status); // keep the caret in place for rapid entry
  } catch (err) {
    toast(String(err).slice(0, 80), true);
  }
}

function focusAdd(status) {
  const input = document.querySelector(`.add[data-status="${status}"] .add-input`);
  if (input) input.focus();
}

function cardEl(card) {
  const node = el("article", "card");
  node.dataset.id = card.id;
  node._card = card; // stash full data for the editor

  node.append(textEl("div", "title", card.title));

  const desc = descriptionPreview(card.body);
  if (desc) node.append(textEl("p", "desc", desc));

  const meta = el("div", "meta");
  meta.append(textEl("span", "id", `#${card.id}`));
  for (const tag of card.tags || []) meta.append(textEl("span", "tag", tag));
  if (card.assignee) meta.append(textEl("span", "who", `@${card.assignee}`));
  node.append(meta);

  node.addEventListener("click", (e) => {
    if (node.classList.contains("editing")) return;
    if (e.target.closest("a")) return;
    openEditor(node);
  });
  return node;
}

// descriptionPreview strips a leading markdown heading that just repeats the
// title, returning the first meaningful lines of the body.
function descriptionPreview(body) {
  if (!body) return "";
  return body
    .split("\n")
    .filter((l) => !/^#{1,6}\s/.test(l))
    .join("\n")
    .trim();
}

// openEditor swaps a card into an inline edit form for title, description,
// tags, and assignee. Saving PUTs to /api/cards/{id}.
function openEditor(node) {
  const card = node._card;
  node.classList.add("editing");
  node.replaceChildren();

  const form = el("form", "editor");
  const title = fieldInput("title", card.title);
  const desc = el("textarea", "f-desc");
  desc.value = descriptionPreview(card.body);
  desc.placeholder = "description (markdown)";
  desc.rows = 4;
  const tags = fieldInput("tags", (card.tags || []).join(", "));
  tags.placeholder = "tags, comma separated";
  const who = fieldInput("assignee", card.assignee || "");
  who.placeholder = "assignee";

  const actions = el("div", "editor-actions");
  const save = button("save", "primary");
  const cancel = button("cancel", "ghost");
  actions.append(save, cancel);

  form.append(labeled("title", title), labeled("description", desc),
    labeled("tags", tags), labeled("assignee", who), actions);
  node.append(form);
  title.focus();

  cancel.addEventListener("click", (e) => { e.preventDefault(); load(); });
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    await update(card.id, {
      title: title.value.trim() || card.title,
      body: buildBody(title.value.trim() || card.title, desc.value),
      tags: splitTags(tags.value),
      assignee: who.value.trim(),
    });
  });
}

// buildBody keeps the card body as a markdown doc with an H1 title followed by
// the description, matching the CLI-created shape.
function buildBody(title, description) {
  const body = description.trim();
  return body ? `# ${title}\n\n${body}\n` : "";
}

function splitTags(raw) {
  return raw.split(",").map((t) => t.trim()).filter(Boolean);
}

async function update(id, fields) {
  try {
    const resp = await fetch(`/api/cards/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(fields),
    });
    if (!resp.ok) throw new Error(await resp.text());
    toast(`saved #${id}`);
    await load();
  } catch (err) {
    toast(String(err).slice(0, 80), true);
  }
}

function makeSortable(list) {
  new Sortable(list, {
    group: "board",
    animation: 160,
    easing: "cubic-bezier(0.22, 1, 0.36, 1)",
    ghostClass: "sortable-ghost",
    chosenClass: "sortable-chosen",
    dragClass: "sortable-drag",
    // Never drag a card that is being edited, nor its form controls.
    filter: ".editing, input, textarea, button",
    preventOnFilter: false,
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
function fieldInput(name, value) {
  const n = el("input", `f-${name}`);
  n.type = "text";
  n.value = value;
  return n;
}
function labeled(label, control) {
  const wrap = el("label", "field");
  wrap.append(textEl("span", "field-label", label), control);
  return wrap;
}
function button(text, variant) {
  const b = el("button", `btn ${variant}`);
  b.type = text === "save" ? "submit" : "button";
  b.textContent = text;
  return b;
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
