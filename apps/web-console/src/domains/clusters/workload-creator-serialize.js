export function serializeVolumeMounts(mounts) {
  return (mounts || [])
    .filter((item) => item.volume_name && item.mount_path)
    .map((item) => `${item.volume_name}:${item.mount_path}:${item.read_only ? "ro" : "rw"}`)
    .join("\n");
}

export function serializeEnvFromSources(items) {
  return (items || [])
    .filter((item) => item.source_type && item.source_name)
    .map((item) => `${item.source_type}:${item.source_name}`)
    .join("\n");
}

export function serializeEnvValueSources(items) {
  return (items || [])
    .filter((item) => item.env_name && item.source_type && item.source_name && item.source_key)
    .map((item) => `${item.env_name}=${item.source_type}:${item.source_name}:${item.source_key}`)
    .join("\n");
}

export function serializeProbe(probe) {
  if (!probe?.enabled) return "";
  const lines = [];
  if (probe.type === "exec") {
    const commands = String(probe.command_text || "")
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean);
    if (commands.length) {
      lines.push("exec:");
      lines.push("  command:");
      commands.forEach((item) => lines.push(`    - ${item}`));
    }
  } else if (probe.type === "tcpSocket") {
    lines.push("tcpSocket:");
    if (probe.host) lines.push(`  host: ${probe.host}`);
    if (probe.port) {
      const portValue = /^\d+$/.test(String(probe.port)) ? Number(probe.port) : probe.port;
      lines.push(`  port: ${portValue}`);
    }
  } else {
    lines.push("httpGet:");
    lines.push(`  path: ${probe.path || "/"}`);
    if (probe.port) {
      const portValue = /^\d+$/.test(String(probe.port)) ? Number(probe.port) : probe.port;
      lines.push(`  port: ${portValue}`);
    }
    if (probe.scheme && probe.scheme !== "HTTP") lines.push(`  scheme: ${probe.scheme}`);
  }
  [["initialDelaySeconds", probe.initial_delay_seconds], ["periodSeconds", probe.period_seconds], ["timeoutSeconds", probe.timeout_seconds], ["successThreshold", probe.success_threshold], ["failureThreshold", probe.failure_threshold]]
    .forEach(([field, value]) => {
      if (value !== "" && value !== undefined && value !== null) lines.push(`${field}: ${Number(value)}`);
    });
  return lines.join("\n");
}

export function serializeContainerSecurityContext(item) {
  if (!item) return "";
  const lines = [];
  if (item.allow_privilege_escalation) lines.push("allowPrivilegeEscalation: true");
  if (item.privileged) lines.push("privileged: true");
  if (item.read_only_root_filesystem) lines.push("readOnlyRootFilesystem: true");
  if (item.run_as_non_root) lines.push("runAsNonRoot: true");
  if (item.run_as_user) lines.push(`runAsUser: ${Number(item.run_as_user)}`);
  if (item.run_as_group) lines.push(`runAsGroup: ${Number(item.run_as_group)}`);
  const add = String(item.capabilities_add_text || "").split("\n").map((line) => line.trim()).filter(Boolean);
  const drop = String(item.capabilities_drop_text || "").split("\n").map((line) => line.trim()).filter(Boolean);
  if (add.length || drop.length) {
    lines.push("capabilities:");
    if (add.length) {
      lines.push("  add:");
      add.forEach((value) => lines.push(`    - ${value}`));
    }
    if (drop.length) {
      lines.push("  drop:");
      drop.forEach((value) => lines.push(`    - ${value}`));
    }
  }
  return lines.join("\n");
}

export function serializeNodeSelectors(items) {
  return (items || [])
    .filter((item) => item.key && item.value)
    .map((item) => `${item.key}=${item.value}`)
    .join("\n");
}

export function serializeTolerations(items) {
  const rows = (items || []).filter((item) => item.key || item.value || item.effect || item.operator === "Exists");
  if (!rows.length) return "";
  return rows.map((item) => {
    const lines = [];
    lines.push(item.key ? `- key: ${item.key}` : `- key: ""`);
    lines.push(`  operator: ${item.operator || "Equal"}`);
    if (item.operator !== "Exists" && item.value) lines.push(`  value: ${item.value}`);
    if (item.effect) lines.push(`  effect: ${item.effect}`);
    if (item.toleration_seconds) lines.push(`  tolerationSeconds: ${Number(item.toleration_seconds)}`);
    return lines.join("\n");
  }).join("\n");
}

export function serializeHostAliases(items) {
  const rows = (items || []).filter((item) => item.ip && item.hostnames_text);
  if (!rows.length) return "";
  return rows.map((item) => {
    const hostnames = String(item.hostnames_text || "")
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean);
    const lines = [`- ip: ${item.ip}`, "  hostnames:"];
    hostnames.forEach((hostname) => lines.push(`    - ${hostname}`));
    return lines.join("\n");
  }).join("\n");
}

export function serializeDNSConfig(builder) {
  const nameservers = String(builder.dns_nameservers_text || "").split("\n").map((line) => line.trim()).filter(Boolean);
  const searches = String(builder.dns_searches_text || "").split("\n").map((line) => line.trim()).filter(Boolean);
  const options = (builder.dns_options || []).filter((item) => item.name);
  if (!nameservers.length && !searches.length && !options.length) return "";
  const lines = [];
  if (nameservers.length) {
    lines.push("nameservers:");
    nameservers.forEach((item) => lines.push(`  - ${item}`));
  }
  if (searches.length) {
    lines.push("searches:");
    searches.forEach((item) => lines.push(`  - ${item}`));
  }
  if (options.length) {
    lines.push("options:");
    options.forEach((item) => {
      lines.push(`  - name: ${item.name}`);
      if (item.value) lines.push(`    value: ${item.value}`);
    });
  }
  return lines.join("\n");
}

export function serializePodSecurityContext(item) {
  if (!item) return "";
  const lines = [];
  if (item.run_as_non_root) lines.push("runAsNonRoot: true");
  if (item.run_as_user) lines.push(`runAsUser: ${Number(item.run_as_user)}`);
  if (item.run_as_group) lines.push(`runAsGroup: ${Number(item.run_as_group)}`);
  if (item.fs_group) lines.push(`fsGroup: ${Number(item.fs_group)}`);
  return lines.join("\n");
}

function serializeNodeAffinity(builder) {
  const required = (builder.node_affinity_required || []).filter((item) => item.key);
  const preferred = (builder.node_affinity_preferred || []).filter((item) => item.key);
  if (!required.length && !preferred.length) return "";
  const lines = ["nodeAffinity:"];
  if (required.length) {
    lines.push("  requiredDuringSchedulingIgnoredDuringExecution:");
    lines.push("    nodeSelectorTerms:");
    lines.push("      - matchExpressions:");
    required.forEach((item) => {
      lines.push(`          - key: ${item.key}`);
      lines.push(`            operator: ${item.operator || "In"}`);
      const values = String(item.values_text || "").split("\n").map((v) => v.trim()).filter(Boolean);
      if (values.length && !["Exists", "DoesNotExist"].includes(item.operator)) {
        lines.push("            values:");
        values.forEach((value) => lines.push(`              - ${value}`));
      }
    });
  }
  if (preferred.length) {
    lines.push("  preferredDuringSchedulingIgnoredDuringExecution:");
    preferred.forEach((item) => {
      lines.push(`    - weight: ${Number(item.weight || 50)}`);
      lines.push("      preference:");
      lines.push("        matchExpressions:");
      lines.push(`          - key: ${item.key}`);
      lines.push(`            operator: ${item.operator || "In"}`);
      const values = String(item.values_text || "").split("\n").map((v) => v.trim()).filter(Boolean);
      if (values.length && !["Exists", "DoesNotExist"].includes(item.operator)) {
        lines.push("            values:");
        values.forEach((value) => lines.push(`              - ${value}`));
      }
    });
  }
  return lines.join("\n");
}

function serializePodAffinitySection(kind, required, preferred) {
  const req = (required || []).filter((item) => item.label_key && item.label_value && item.topology_key);
  const pref = (preferred || []).filter((item) => item.label_key && item.label_value && item.topology_key);
  if (!req.length && !pref.length) return [];
  const lines = [`${kind}:`];
  if (req.length) {
    lines.push("  requiredDuringSchedulingIgnoredDuringExecution:");
    req.forEach((item) => {
      lines.push("    - labelSelector:");
      lines.push("        matchExpressions:");
      lines.push(`          - key: ${item.label_key}`);
      lines.push("            operator: In");
      lines.push("            values:");
      lines.push(`              - ${item.label_value}`);
      lines.push(`      topologyKey: ${item.topology_key}`);
    });
  }
  if (pref.length) {
    lines.push("  preferredDuringSchedulingIgnoredDuringExecution:");
    pref.forEach((item) => {
      lines.push(`    - weight: ${Number(item.weight || 80)}`);
      lines.push("      podAffinityTerm:");
      lines.push("        labelSelector:");
      lines.push("          matchExpressions:");
      lines.push(`            - key: ${item.label_key}`);
      lines.push("              operator: In");
      lines.push("              values:");
      lines.push(`                - ${item.label_value}`);
      lines.push(`        topologyKey: ${item.topology_key}`);
    });
  }
  return lines;
}

export function serializeAffinity(builder) {
  const sections = [];
  const nodeAffinity = serializeNodeAffinity(builder);
  if (nodeAffinity) sections.push(nodeAffinity);
  const podAffinity = serializePodAffinitySection("podAffinity", builder.pod_affinity_required, builder.pod_affinity_preferred);
  if (podAffinity.length) sections.push(...podAffinity);
  const podAntiAffinity = serializePodAffinitySection("podAntiAffinity", builder.pod_anti_affinity_required, builder.pod_anti_affinity_preferred);
  if (podAntiAffinity.length) sections.push(...podAntiAffinity);
  return sections.length ? sections.join("\n") : "";
}

export function serializeTopologySpreadConstraints(items) {
  const rows = (items || []).filter((item) => item.max_skew && item.topology_key && item.when_unsatisfiable && item.label_key && item.label_value);
  if (!rows.length) return "";
  return rows.map((item) => [
    `- maxSkew: ${Number(item.max_skew)}`,
    `  topologyKey: ${item.topology_key}`,
    `  whenUnsatisfiable: ${item.when_unsatisfiable}`,
    "  labelSelector:",
    "    matchLabels:",
    `      ${item.label_key}: ${item.label_value}`,
  ].join("\n")).join("\n");
}
