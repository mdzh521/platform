# Web Console UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the `apps/web-console` product shell, overview page, and first-screen module layouts into a more formal SaaS-style operations console without breaking current module wiring.

**Architecture:** Keep the existing module IDs, data mount points, and event targets intact while restructuring the static HTML shell and swapping the visual system in `assets/styles.css`. Limit behavior changes to title/copy updates in the boot/page entrypoints so existing domain loaders continue to render into the same anchors.

**Tech Stack:** Static HTML, modular browser JavaScript, shared CSS, Node-based structure verification

---

### Task 1: Lock the new shell contract with structure tests

**Files:**
- Modify: `apps/web-console/test/verify-structure.mjs`
- Test: `apps/web-console/test/verify-structure.mjs`

- [ ] **Step 1: Add failing assertions for the renamed primary navigation and overview shell anchors**

```js
  for (const label of ["总览", "交付网络", "集群工作台", "机器工作台", "系统治理"]) {
    assert(indexHtml.includes(`>${label}<`), `index.html must include primary nav label: ${label}`);
  }
  for (const id of [
    "overview-ops-grid",
    "overview-activity-grid",
    "delivery-page-header",
    "clusters-page-header",
    "machine-page-header",
    "system-page-header",
  ]) {
    assert(indexHtml.includes(`id=\"${id}\"`), `index.html must include redesigned shell anchor: ${id}`);
  }
```

- [ ] **Step 2: Run the structure test and verify it fails**

Run: `node test/verify-structure.mjs`
Expected: FAIL with a missing nav label or missing redesigned shell anchor

- [ ] **Step 3: Keep the failing assertions in place as the acceptance contract for the HTML rewrite**

No additional code beyond Step 1.

- [ ] **Step 4: Re-run after implementation and verify it passes**

Run: `node test/verify-structure.mjs`
Expected: PASS with no output

### Task 2: Rebuild the global product shell and overview page

**Files:**
- Modify: `apps/web-console/index.html`
- Modify: `apps/web-console/src/pages/overview/index.js`
- Test: `apps/web-console/test/verify-structure.mjs`

- [ ] **Step 1: Restructure the global header and primary navigation in `index.html`**

Keep the existing `data-top-module` values, but update labels and shell copy:

```html
<header class="topbar">
  <div class="topbar-left">
    <div class="brand-mark small">PC</div>
    <div>
      <p class="eyebrow">Platform Center</p>
      <h1 id="module-title">总览</h1>
      <p class="topbar-context">统一查看交付、集群、机器与治理状态。</p>
    </div>
  </div>
  <div class="topbar-meta">
    <span id="permission-badge" class="permission-badge">0 项权限</span>
    <span id="current-user-name" class="current-user-name">-</span>
    <button id="logout-button" class="ghost-button">退出</button>
  </div>
</header>

<nav class="top-tabs">
  <button class="top-tab active" data-top-module="overview">总览</button>
  <button class="top-tab" data-top-module="cloud">交付网络</button>
  <button class="top-tab" data-top-module="business">集群工作台</button>
  <button class="top-tab" data-top-module="machine">机器工作台</button>
  <button class="top-tab" data-top-module="system">系统治理</button>
</nav>
```

- [ ] **Step 2: Rewrite the overview section into hero + metrics + work domains + activity lanes**

Add the new anchor containers while preserving `overview-summary-grid`:

```html
<section id="top-module-overview" class="top-module">
  <section class="overview-shell">
    <div class="page-header page-header-overview">
      ...
    </div>
    <div id="overview-summary-grid" class="overview-summary-grid"></div>
    <div id="overview-ops-grid" class="overview-ops-grid">...</div>
    <div id="overview-activity-grid" class="overview-activity-grid">...</div>
  </section>
</section>
```

- [ ] **Step 3: Update `src/pages/overview/index.js` to render summary copy that matches the new IA**

Keep the same metrics sources, but change labels to the new domains:

```js
    <article class="overview-summary-card">
      <span class="muted-label">交付网络</span>
      <strong>${clusters}</strong>
      <p>已接入集群 ${clusters} 个，当前工作负载 ${workloads} 个。</p>
    </article>
```

- [ ] **Step 4: Run the structure test and verify the overview shell contract passes**

Run: `node test/verify-structure.mjs`
Expected: PASS

### Task 3: Reframe the four top-level modules without changing domain mount points

**Files:**
- Modify: `apps/web-console/index.html`
- Modify: `apps/web-console/src/boot/app.js`
- Modify: `apps/web-console/src/pages/system/index.js`
- Test: `apps/web-console/test/verify-structure.mjs`

- [ ] **Step 1: Add page-header wrappers for delivery, clusters, machines, and system**

Use new IDs for the headers while keeping existing inner mount points:

```html
<div id="delivery-page-header" class="page-header">...</div>
<div id="clusters-page-header" class="page-header">...</div>
<div id="machine-page-header" class="page-header">...</div>
<div id="system-page-header" class="page-header">...</div>
```

- [ ] **Step 2: Update module titles in `src/boot/app.js`**

```js
  const titles = {
    overview: "总览",
    cloud: "交付网络",
    business: "集群工作台",
    system: "系统治理",
    machine: "机器工作台",
  };
```

- [ ] **Step 3: Update nested system module titles in `src/pages/system/index.js`**

```js
    elements.moduleTitle.textContent = "系统治理 / 用户与身份";
```

- [ ] **Step 4: Re-run the structure test after the module-shell rewrite**

Run: `node test/verify-structure.mjs`
Expected: PASS

### Task 4: Replace the visual system and verify the redesigned shell renders

**Files:**
- Modify: `apps/web-console/assets/styles.css`
- Test: `apps/web-console/test/verify-structure.mjs`

- [ ] **Step 1: Replace the root tokens and global shell styling**

Update `:root`, `body`, `.workspace`, `.topbar`, `.top-tabs`, `.page-header`, `.panel`, and button styles to establish the new SaaS control-console look.

- [ ] **Step 2: Add specific layout styling for overview, delivery, clusters, machines, and system**

Cover:
- `.overview-ops-grid`
- `.overview-activity-grid`
- `.cloud-stage-shell`
- `.business-hero`
- `.machine-hero`
- `.system-layout`

- [ ] **Step 3: Run the structure test after the CSS rewrite**

Run: `node test/verify-structure.mjs`
Expected: PASS

- [ ] **Step 4: Open the local app in the browser and verify the redesigned shell visually**

Use the browser plugin against the local web-console URL once the app is available.
Expected: The header, nav, overview, and top-level module shells reflect the approved IA and visual hierarchy.
