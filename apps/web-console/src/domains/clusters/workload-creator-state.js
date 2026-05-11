import { state } from "../../core/state.js";

export function createDefaultBuilderState(context, preset = "Deployment") {
  const clusterID = context.selectedWorkloadClusterID() || context.selectedClusterID() || state.clusters[0]?.id || "";
  const namespace = context.selectedWorkloadNamespace() || state.business.selectedNamespace || "";
  return {
    cluster_id: clusterID ? String(clusterID) : "",
    namespace,
    workload_type: preset,
    name: "",
    labels_text: "",
    annotations_text: "",
    replicas: 1,
    service_name: "",
    pod_management_policy: "OrderedReady",
    statefulset_service_mode: "Headless",
    statefulset_partition: 0,
    schedule: "*/5 * * * *",
    suspend: false,
    concurrency_policy: "Allow",
    starting_deadline_seconds: 0,
    successful_jobs_history_limit: 3,
    failed_jobs_history_limit: 1,
    create_service_account: false,
    service_account_name: "",
    update_strategy: "RollingUpdate",
    max_surge: "25%",
    max_unavailable: "25%",
    node_selector_text: "",
    node_selectors: [],
    tolerations_yaml: "",
    tolerations: [],
    affinity_yaml: "",
    node_affinity_required: [],
    node_affinity_preferred: [],
    pod_affinity_required: [],
    pod_affinity_preferred: [],
    pod_anti_affinity_required: [],
    pod_anti_affinity_preferred: [],
    dns_policy: "ClusterFirst",
    dns_config_yaml: "",
    dns_nameservers_text: "",
    dns_searches_text: "",
    dns_options: [],
    host_aliases_yaml: "",
    host_aliases: [],
    host_network: false,
    host_pid: false,
    host_ipc: false,
    image_pull_secrets_text: "",
    topology_spread_yaml: "",
    topology_spread_constraints: [],
    pod_security_context_yaml: "",
    pod_security_context: {
      run_as_non_root: false,
      run_as_user: "",
      run_as_group: "",
      fs_group: "",
    },
    termination_grace_period_seconds: 30,
    create_service: ["Deployment", "StatefulSet", "DaemonSet"].includes(preset),
    service_type: "ClusterIP",
    service_port: 80,
    create_ingress: false,
    ingress_host: "",
    ingress_path: "/",
    ingress_class_name: "",
    containers: [
      createEmptyContainer("main", "nginx:1.27"),
    ],
    init_containers: [],
    volumes: [],
    claim_templates: [],
  };
}

export function createDefaultYAMLState(context, preset = "Deployment") {
  const clusterID = context.selectedWorkloadClusterID() || context.selectedClusterID() || state.clusters[0]?.id || "";
  const namespace = context.selectedWorkloadNamespace() || state.business.selectedNamespace || "";
  return {
    cluster_id: clusterID ? String(clusterID) : "",
    namespace,
    manifest_yaml: buildManifestTemplate(preset, namespace),
  };
}

export function buildManifestTemplate(preset, namespace = "") {
  const ns = namespace ? `  namespace: ${namespace}\n` : "";
  switch (preset) {
    case "Service":
      return `apiVersion: v1\nkind: Service\nmetadata:\n  name: demo-service\n${ns}spec:\n  selector:\n    app: demo-app\n  ports:\n    - port: 80\n      targetPort: 8080\n  type: ClusterIP\n`;
    case "Ingress":
      return `apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: demo-ingress\n${ns}spec:\n  ingressClassName: nginx\n  rules:\n    - host: demo.local\n      http:\n        paths:\n          - path: /\n            pathType: Prefix\n            backend:\n              service:\n                name: demo-service\n                port:\n                  number: 80\n`;
    case "ConfigMap":
      return `apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: demo-config\n${ns}data:\n  APP_ENV: production\n  LOG_LEVEL: info\n`;
    case "Secret":
      return `apiVersion: v1\nkind: Secret\nmetadata:\n  name: demo-secret\n${ns}type: Opaque\nstringData:\n  username: admin\n  password: change-me\n`;
    case "PersistentVolumeClaim":
      return `apiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  name: demo-pvc\n${ns}spec:\n  accessModes:\n    - ReadWriteOnce\n  resources:\n    requests:\n      storage: 5Gi\n`;
    case "ServiceAccount":
      return `apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: demo-serviceaccount\n${ns}imagePullSecrets:\n  - name: regcred\n`;
    case "ResourceQuota":
      return `apiVersion: v1\nkind: ResourceQuota\nmetadata:\n  name: demo-quota\n${ns}spec:\n  hard:\n    pods: "10"\n    requests.cpu: "2"\n    requests.memory: 2Gi\n`;
    case "LimitRange":
      return `apiVersion: v1\nkind: LimitRange\nmetadata:\n  name: demo-limits\n${ns}spec:\n  limits:\n    - type: Container\n      default:\n        cpu: "500m"\n        memory: 512Mi\n      defaultRequest:\n        cpu: "100m"\n        memory: 128Mi\n`;
    case "CronJob":
      return `apiVersion: batch/v1\nkind: CronJob\nmetadata:\n  name: demo-cron\n${ns}spec:\n  schedule: "*/5 * * * *"\n  jobTemplate:\n    spec:\n      template:\n        spec:\n          restartPolicy: OnFailure\n          containers:\n            - name: demo-cron\n              image: busybox:1.36\n              command: ["/bin/sh","-c","date; echo hello"]\n`;
    default:
      return `apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: demo-app\n${ns}spec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: demo-app\n  template:\n    metadata:\n      labels:\n        app: demo-app\n    spec:\n      containers:\n        - name: demo-app\n          image: nginx:1.27\n          ports:\n            - containerPort: 80\n`;
  }
}

export function ensureWorkloadCreatorDefaults() {
  if (!state.workloadCreator.builder) {
    throw new Error("workload creator builder state is not initialized");
  }
  if (!state.workloadCreator.yaml) {
    throw new Error("workload creator yaml state is not initialized");
  }
  if (!Array.isArray(state.workloadCreator.builder.node_selectors)) state.workloadCreator.builder.node_selectors = [];
  if (!Array.isArray(state.workloadCreator.builder.tolerations)) state.workloadCreator.builder.tolerations = [];
  if (!Array.isArray(state.workloadCreator.builder.node_affinity_required)) state.workloadCreator.builder.node_affinity_required = [];
  if (!Array.isArray(state.workloadCreator.builder.node_affinity_preferred)) state.workloadCreator.builder.node_affinity_preferred = [];
  if (!Array.isArray(state.workloadCreator.builder.pod_affinity_required)) state.workloadCreator.builder.pod_affinity_required = [];
  if (!Array.isArray(state.workloadCreator.builder.pod_affinity_preferred)) state.workloadCreator.builder.pod_affinity_preferred = [];
  if (!Array.isArray(state.workloadCreator.builder.pod_anti_affinity_required)) state.workloadCreator.builder.pod_anti_affinity_required = [];
  if (!Array.isArray(state.workloadCreator.builder.pod_anti_affinity_preferred)) state.workloadCreator.builder.pod_anti_affinity_preferred = [];
  if (!Array.isArray(state.workloadCreator.builder.host_aliases)) state.workloadCreator.builder.host_aliases = [];
  if (!Array.isArray(state.workloadCreator.builder.topology_spread_constraints)) state.workloadCreator.builder.topology_spread_constraints = [];
  if (!Array.isArray(state.workloadCreator.builder.claim_templates)) state.workloadCreator.builder.claim_templates = [];
  if (!Array.isArray(state.workloadCreator.builder.dns_options)) state.workloadCreator.builder.dns_options = [];
  if (!state.workloadCreator.builder.pod_security_context) {
    state.workloadCreator.builder.pod_security_context = {
      run_as_non_root: false,
      run_as_user: "",
      run_as_group: "",
      fs_group: "",
    };
  }
  ["containers", "init_containers"].forEach((listKey) => {
    const list = state.workloadCreator.builder[listKey] || [];
    list.forEach((item) => {
      if (!Array.isArray(item.volume_mounts)) item.volume_mounts = [];
      if (!Array.isArray(item.env_from_sources)) item.env_from_sources = [];
      if (!Array.isArray(item.env_value_sources)) item.env_value_sources = [];
      if (!item.liveness_probe) item.liveness_probe = createEmptyProbe();
      if (!item.readiness_probe) item.readiness_probe = createEmptyProbe();
      if (!item.startup_probe) item.startup_probe = createEmptyProbe();
      if (!item.security_context) item.security_context = createEmptyContainerSecurityContext();
    });
  });
}

export function createEmptyClaimTemplate() {
  return {
    name: "",
    storage_class_name: "",
    size: "1Gi",
    access_modes_text: "ReadWriteOnce",
    labels_text: "",
    annotations_text: "",
  };
}

export function createEmptyContainer(name = "", image = "") {
  return {
    name,
    image,
    ports_text: "",
    command_text: "",
    args_text: "",
    env_text: "",
    env_from_text: "",
    env_from_sources: [],
    env_value_from_text: "",
    env_value_sources: [],
    image_pull_policy: "IfNotPresent",
    working_dir: "",
    stdin: false,
    tty: false,
    lifecycle_post_start_text: "",
    lifecycle_pre_stop_text: "",
    volume_mounts_text: "",
    volume_mounts: [],
    cpu_request: "",
    memory_request: "",
    cpu_limit: "",
    memory_limit: "",
    liveness_probe_yaml: "",
    liveness_probe: createEmptyProbe(),
    readiness_probe_yaml: "",
    readiness_probe: createEmptyProbe(),
    startup_probe_yaml: "",
    startup_probe: createEmptyProbe(),
    security_context: createEmptyContainerSecurityContext(),
    security_context_yaml: "",
  };
}

export function createEmptyVolumeMount() {
  return {
    volume_name: "",
    mount_path: "",
    read_only: false,
  };
}

export function createEmptyEnvFromSource() {
  return {
    source_type: "configMap",
    source_name: "",
  };
}

export function createEmptyEnvValueSource() {
  return {
    env_name: "",
    source_type: "configMap",
    source_name: "",
    source_key: "",
  };
}

export function createEmptyProbe() {
  return {
    enabled: false,
    type: "httpGet",
    path: "/",
    port: "80",
    scheme: "HTTP",
    command_text: "",
    host: "",
    initial_delay_seconds: "",
    period_seconds: "",
    timeout_seconds: "",
    success_threshold: "",
    failure_threshold: "",
  };
}

export function createEmptyContainerSecurityContext() {
  return {
    allow_privilege_escalation: false,
    privileged: false,
    read_only_root_filesystem: false,
    run_as_non_root: false,
    run_as_user: "",
    run_as_group: "",
    capabilities_add_text: "",
    capabilities_drop_text: "",
  };
}

export function createEmptyNodeSelector() {
  return { key: "", value: "" };
}

export function createEmptyTopologySpreadConstraint() {
  return {
    max_skew: "1",
    topology_key: "kubernetes.io/hostname",
    when_unsatisfiable: "ScheduleAnyway",
    label_key: "app",
    label_value: "",
  };
}

export function createEmptyToleration() {
  return {
    key: "",
    operator: "Equal",
    value: "",
    effect: "",
    toleration_seconds: "",
  };
}

export function createEmptyHostAlias() {
  return {
    ip: "",
    hostnames_text: "",
  };
}

export function createEmptyDNSOption() {
  return { name: "", value: "" };
}

export function createEmptyNodeAffinityRule(weight = "") {
  return {
    key: "",
    operator: "In",
    values_text: "",
    weight,
  };
}

export function createEmptyPodAffinityRule(weight = "", topologyKey = "kubernetes.io/hostname") {
  return {
    label_key: "",
    label_value: "",
    topology_key: topologyKey,
    weight,
  };
}

export function normalizeCreatorValue(key, value) {
  if (["replicas", "service_port", "termination_grace_period_seconds", "starting_deadline_seconds", "successful_jobs_history_limit", "failed_jobs_history_limit"].includes(key)) {
    return Number(value || 0);
  }
  if (["suspend", "create_service_account", "create_service", "create_ingress", "read_only", "run_as_non_root", "enabled", "allow_privilege_escalation", "privileged", "read_only_root_filesystem", "stdin", "tty", "host_network", "host_pid", "host_ipc"].includes(key)) {
    return value === "true";
  }
  return value;
}
