const fallbackState = {
  status: "idle",
  message: "Press Enter to record",
  error: "",
  text: "",
  ready: false,
};

const statusEl = document.getElementById("status");
const detailEl = document.getElementById("detail");
const meterFillEl = document.getElementById("meter-fill");
const primaryEl = document.getElementById("primary");
const secondaryEl = document.getElementById("secondary");

let currentState = { ...fallbackState };
let statusTimer = null;
let levelTimer = null;

function guiBindings() {
  return window?.go?.gui?.App ?? null;
}

function updateView(state) {
  const next = state || fallbackState;
  currentState = next;
  statusEl.textContent = next.message || "mestt";

  if (next.error) {
    detailEl.textContent = next.error;
  } else if (next.text && next.status === "copied") {
    detailEl.textContent = next.text;
  } else if (next.ready) {
    detailEl.textContent = detailForState(next.status);
  } else {
    detailEl.textContent = "Waiting for Wails runtime...";
  }

  updateButtons(next.status);
  if (next.status !== "recording") {
    meterFillEl.style.width = `${Math.round(idleMeter(next.status) * 100)}%`;
  }
  ensurePolling(next.status);
}

function detailForState(status) {
  switch (status) {
    case "preparing":
      return "Checking local model and dependencies before recording starts.";
    case "recording":
      return "Recording microphone audio. Press Enter again to stop.";
    case "transcribing":
      return "Turning the captured audio into text and copying it to the clipboard.";
    case "copied":
      return "Transcript copied to the clipboard.";
    case "error":
      return "Something went wrong.";
    default:
      return "Ready to record.";
  }
}

function updateButtons(status) {
  primaryEl.disabled = false;
  secondaryEl.disabled = false;

  switch (status) {
    case "preparing":
      primaryEl.textContent = "Preparing...";
      primaryEl.disabled = true;
      secondaryEl.textContent = "Cancel";
      break;
    case "recording":
      primaryEl.textContent = "Stop";
      secondaryEl.textContent = "Cancel";
      break;
    case "transcribing":
      primaryEl.textContent = "Transcribing...";
      primaryEl.disabled = true;
      secondaryEl.textContent = "Cancel";
      break;
    case "copied":
      primaryEl.textContent = "Record Again";
      secondaryEl.textContent = "Close";
      break;
    case "error":
      primaryEl.textContent = "Retry";
      secondaryEl.textContent = "Dismiss";
      break;
    default:
      primaryEl.textContent = "Record";
      secondaryEl.textContent = "Close";
      break;
  }
}

function idleMeter(status) {
  if (status === "preparing") return 0.42;
  if (status === "transcribing") return 0.58;
  if (status === "copied") return 1;
  if (status === "error") return 0.22;
  return 0.18;
}

function ensurePolling(status) {
  if (statusTimer) {
    clearInterval(statusTimer);
    statusTimer = null;
  }
  if (levelTimer) {
    clearInterval(levelTimer);
    levelTimer = null;
  }

  if (status === "preparing" || status === "transcribing") {
    statusTimer = window.setInterval(refreshStatus, 250);
  }
  if (status === "recording") {
    statusTimer = window.setInterval(refreshStatus, 250);
    levelTimer = window.setInterval(refreshLevels, 120);
  }
}

async function refreshStatus() {
  const bindings = guiBindings();
  if (!bindings?.Status) {
    updateView(fallbackState);
    return;
  }
  try {
    updateView(await bindings.Status());
  } catch (error) {
    updateView({ ...fallbackState, status: "error", error: error.message });
  }
}

async function refreshLevels() {
  const bindings = guiBindings();
  if (!bindings?.Levels) {
    return;
  }
  try {
    const levels = await bindings.Levels();
    const fill = Number.isFinite(levels.level) ? levels.level : 0;
    meterFillEl.style.width = `${Math.max(0, Math.min(100, Math.round(fill)))}%`;
  } catch (_) {
    // Ignore transient level polling failures.
  }
}

async function primaryAction() {
  const bindings = guiBindings();
  if (!bindings) {
    updateView(fallbackState);
    return;
  }

  switch (currentState.status) {
    case "recording":
      updateView(await bindings.StopAndTranscribe());
      break;
    case "copied":
    case "error":
      updateView(await bindings.DismissError());
      updateView(await bindings.StartRecording());
      break;
    case "preparing":
    case "transcribing":
      break;
    default:
      updateView(await bindings.StartRecording());
      break;
  }
}

async function dismiss() {
  const bindings = guiBindings();
  if (currentState.status === "preparing" || currentState.status === "recording" || currentState.status === "transcribing") {
    if (bindings?.CancelRecording) {
      updateView(await bindings.CancelRecording());
    }
    return;
  }
  if (bindings?.DismissError) {
    updateView(await bindings.DismissError());
    return;
  }
  updateView(fallbackState);
}

primaryEl.addEventListener("click", primaryAction);
secondaryEl.addEventListener("click", dismiss);

document.addEventListener("keydown", async (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    await primaryAction();
  }
  if (event.key === "Escape" || (event.ctrlKey && event.key.toLowerCase() === "c")) {
    event.preventDefault();
    await dismiss();
  }
});

window.addEventListener("beforeunload", () => {
  if (statusTimer) {
    clearInterval(statusTimer);
  }
  if (levelTimer) {
    clearInterval(levelTimer);
  }
});

refreshStatus();
