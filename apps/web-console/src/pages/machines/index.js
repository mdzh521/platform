import { activateMachineWorkspace } from "../../domains/machines/index.js";

export function enterMachinesPage() {
  window.requestAnimationFrame(() => activateMachineWorkspace());
}
