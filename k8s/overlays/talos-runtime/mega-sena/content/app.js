const megaSenaDraws = Array.isArray(window.megaSenaDraws)
  ? window.megaSenaDraws.map((draw) => ({
      ...draw,
      ordemSorteio: Array.isArray(draw.ordemSorteio) ? draw.ordemSorteio : null,
    }))
  : [];

const sortedAxisLabels = [
  "1ª dezena",
  "2ª dezena",
  "3ª dezena",
  "4ª dezena",
  "5ª dezena",
  "6ª dezena",
];

const orderedAxisLabels = [
  "1ª sorteada",
  "2ª sorteada",
  "3ª sorteada",
  "4ª sorteada",
  "5ª sorteada",
  "6ª sorteada",
];

const totalDrawsEl = document.getElementById("total-draws");
const firstDrawEl = document.getElementById("first-draw");
const lastDrawEl = document.getElementById("last-draw");
const dateRangeEl = document.getElementById("date-range");
const detailTitleEl = document.getElementById("detail-title");
const detailMetaEl = document.getElementById("detail-meta");
const detailGridSortedEl = document.getElementById("detail-grid-sorted");
const detailGridOrderedEl = document.getElementById("detail-grid-ordered");
const detailOrderMetaEl = document.getElementById("detail-order-meta");
const highlightNumberEl = document.getElementById("highlight-number");
const highlightRangeEl = document.getElementById("highlight-range");
const opacityRangeEl = document.getElementById("opacity-range");
const toggleAllLinesButton = document.getElementById("toggle-all-lines");
const rangePanelEl = document.getElementById("range-panel");
const rangeBarEl = document.querySelector(".dual-range");
const rangeCountEl = document.getElementById("range-count");
const rangeStartLabelEl = document.getElementById("range-start-label");
const rangeEndLabelEl = document.getElementById("range-end-label");
const rangeFillEl = document.getElementById("range-fill");
const rangeStartEl = document.getElementById("range-start");
const rangeHighlightEl = document.getElementById("range-highlight");
const rangeEndEl = document.getElementById("range-end");
const dateStartEl = document.getElementById("date-start");
const dateEndEl = document.getElementById("date-end");
const backgroundToneRangeEl = document.getElementById("background-tone-range");
const backgroundToneLabelEl = document.getElementById("background-tone-label");
const backgroundToneSwatchEl = document.getElementById("background-tone-swatch");
const groupFilterEnabledEl = document.getElementById("group-filter-enabled");
const groupFilterSourceEl = document.getElementById("group-filter-source");
const groupFilterSizeEl = document.getElementById("group-filter-size");
const groupFilterModeEl = document.getElementById("group-filter-mode");
const groupFilterKeyEl = document.getElementById("group-filter-key");
const groupFilterMetaEl = document.getElementById("group-filter-meta");
const recentOverlayLimitEl = document.getElementById("recent-overlay-limit");
const recentOverlayMetaEl = document.getElementById("recent-overlay-meta");
const recentMonthStartEl = document.getElementById("recent-month-start");
const recentMonthEndEl = document.getElementById("recent-month-end");
const simulationInputs = [...document.querySelectorAll(".simulation-input")];
const addSimulationButton = document.getElementById("add-simulation");
const autoSimulationButton = document.getElementById("auto-simulation");
const clearSimulationsButton = document.getElementById("clear-simulations");
const simulationMetaEl = document.getElementById("simulation-meta");
const simulationListEl = document.getElementById("simulation-list");
const updateMegaSenaResultButton = document.getElementById("update-mega-sena-result");
const updateMegaSenaStatusEl = document.getElementById("update-mega-sena-status");
const sorteGameBoardEl = document.getElementById("sorte-game-board");
const sorteGameStatusEl = document.getElementById("sorte-game-status");
const sorteGameRegisterButton = document.getElementById("sorte-game-register");
const sorteGameResetButton = document.getElementById("sorte-game-reset");
const sorteGameRegisterStatusEl = document.getElementById("sorte-game-register-status");
const exportPngButton = document.getElementById("export-png");

const clamp = (value, min, max) => Math.min(max, Math.max(min, value));
const formatNumber = (value) => String(value).padStart(2, "0");
const getDateKey = (draw) => draw.dataSorteio.slice(0, 10);
const getMonthKey = (draw) => draw.dataSorteio.slice(0, 7);
const MEGA_SENA_UPDATE_ENDPOINT = "/api/mega-sena/ultimo-resultado";
const MEGA_SENA_SIMULATION_ENDPOINT = "/api/mega-sena/simulacoes";
const sorteGameState = {
  slots: Array.from({ length: 6 }, () => ({ value: null, drawnAt: null })),
  nextIndex: 0,
  animatingIndex: null,
  timers: [],
  savedSequence: null,
  registerPending: false,
};

const formatDate = (value) =>
  new Intl.DateTimeFormat("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    timeZone: "UTC",
  }).format(new Date(value));

const drawMap = new Map(megaSenaDraws.map((draw, index) => [draw.concurso, index]));

const state = {
  highlightIndex: Math.max(0, megaSenaDraws.length - 1),
  baseOpacity: Number(opacityRangeEl.value) / 100,
  showAllLines: false,
  rangeStartIndex: 0,
  rangeEndIndex: Math.max(0, megaSenaDraws.length - 1),
  overlayBackgroundTone: 210,
  overlayGradientStart: "#ffffff",
  groupFilterEnabled: false,
  groupFilterSource: "ordered",
  groupFilterSize: 2,
  groupFilterMode: "set",
  groupFilterKey: "",
  recentOverlayLimit: 0,
  recentMonthStart: "",
  recentMonthEnd: "",
  simulatedGames: [],
  isAutoChoosing: false,
  autoChoiceRun: 0,
};

const chartConfigs = [
  {
    key: "sorted",
    title: "ordem crescente",
    axisLabels: sortedAxisLabels,
    shellEl: document.getElementById("sorted-chart-shell"),
    canvasEl: document.getElementById("sorted-chart"),
    noteEl: document.getElementById("sorted-chart-note"),
    primaryValuesFor: (draw) => draw.dezenas,
    comparisonValuesFor: (draw) => draw.ordemSorteio,
    primarySeries: {
      color: "blue",
      label: "Azul (ordem crescente)",
      missingLabel: "indisponivel",
    },
    comparisonSeries: {
      color: "red",
      label: "Vermelho (ordem do sorteio)",
      missingLabel: "indisponivel neste concurso",
    },
    unavailableMessage: "Sem dezenas registradas para o concurso em destaque.",
  },
  {
    key: "ordered",
    title: "ordem do sorteio",
    axisLabels: orderedAxisLabels,
    shellEl: document.getElementById("ordered-chart-shell"),
    canvasEl: document.getElementById("ordered-chart"),
    noteEl: document.getElementById("ordered-chart-note"),
    primaryValuesFor: (draw) => draw.ordemSorteio,
    comparisonValuesFor: (draw) => draw.dezenas,
    primarySeries: {
      color: "red",
      label: "Vermelho (ordem do sorteio)",
      missingLabel: "indisponivel neste concurso",
    },
    comparisonSeries: {
      color: "blue",
      label: "Azul (ordem crescente)",
      missingLabel: "indisponivel",
    },
    unavailableMessage:
      "A ordem de sorteio ainda nao esta registrada para este concurso na base local.",
  },
];

chartConfigs.forEach((config) => {
  config.ctx = config.canvasEl.getContext("2d");
});

const mixHexWithWhite = (hex, ratio = 0.68) => {
  const normalized = hex.replace("#", "");
  const safeHex =
    normalized.length === 3
      ? normalized
          .split("")
          .map((char) => `${char}${char}`)
          .join("")
      : normalized.padEnd(6, "f").slice(0, 6);
  const red = Number.parseInt(safeHex.slice(0, 2), 16);
  const green = Number.parseInt(safeHex.slice(2, 4), 16);
  const blue = Number.parseInt(safeHex.slice(4, 6), 16);
  const mix = (channel) => Math.round(channel * (1 - ratio) + 255 * ratio);
  return `rgb(${mix(red)} ${mix(green)} ${mix(blue)})`;
};

const getOverlayBackgroundEnd = (tone = state.overlayBackgroundTone) => {
  const hue = Math.round(tone);
  return `hsl(${hue} 48% 93%)`;
};

const getOverlayBackgroundStart = (color = state.overlayGradientStart) =>
  mixHexWithWhite(color);

const describeOverlayBackground = (tone = state.overlayBackgroundTone) =>
  `${Math.round(tone)}°`;

const formatSequence = (values) => values.map(formatNumber).join(" - ");
const groupSourceLabels = {
  ordered: "ordem do sorteio",
  sorted: "ordem crescente",
  descending: "ordem decrescente",
};

const groupModeLabels = {
  set: "conjunto",
  sequence: "sequencia exata",
};

const isForbiddenOverlayHue = (hue) =>
  hue <= 16 || hue >= 344 || (hue >= 198 && hue <= 255);

const resolveOverlayHue = (seed) => {
  let hue = (seed * 137.508) % 360;
  let attempts = 0;

  while (isForbiddenOverlayHue(hue) && attempts < 12) {
    hue = (hue + 47) % 360;
    attempts += 1;
  }

  return Math.round(hue);
};

const getOverlayLineColor = (draw, alpha = 0.72, point = false) => {
  const hue = resolveOverlayHue(draw.concurso);
  const saturation = 66 + (draw.concurso % 3) * 4;
  const lightness = point ? 34 + (draw.concurso % 2) * 5 : 43 + (draw.concurso % 2) * 5;
  return `hsla(${hue}, ${saturation}%, ${lightness}%, ${alpha})`;
};

const getGroupSourceValues = (draw, source) => {
  if (source === "ordered") {
    return draw.ordemSorteio;
  }

  if (source === "descending") {
    return Array.isArray(draw.dezenas) ? [...draw.dezenas].reverse() : null;
  }

  return draw.dezenas;
};

const getGroupKeyFromValues = (values, size, mode) => {
  if (!Array.isArray(values) || values.length < size) {
    return null;
  }

  const slice = values.slice(0, size);
  const normalized = mode === "set" ? [...slice].sort((a, b) => a - b) : slice;
  return normalized.map(formatNumber).join("-");
};

const getGroupKeyForDraw = (draw, source, size, mode) =>
  getGroupKeyFromValues(getGroupSourceValues(draw, source), size, mode);

const buildRepeatedGroupCatalog = () => {
  const catalog = new Map();
  const sources = ["ordered", "sorted", "descending"];
  const sizes = [2, 3, 4, 5, 6];
  const modes = ["set", "sequence"];

  sources.forEach((source) => {
    sizes.forEach((size) => {
      modes.forEach((mode) => {
        const groups = new Map();

        megaSenaDraws.forEach((draw) => {
          const key = getGroupKeyForDraw(draw, source, size, mode);
          if (!key) {
            return;
          }
          groups.set(key, (groups.get(key) ?? 0) + 1);
        });

        const repeated = [...groups.entries()]
          .filter(([, count]) => count > 1)
          .map(([key, count]) => ({ key, count }))
          .sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));

        catalog.set(`${source}:${size}:${mode}`, repeated);
      });
    });
  });

  return catalog;
};

const repeatedGroupCatalog = buildRepeatedGroupCatalog();

const setupEmptyPage = () => {
  totalDrawsEl.textContent = "0";
  firstDrawEl.textContent = "-";
  lastDrawEl.textContent = "-";
  dateRangeEl.textContent = "-";
  detailTitleEl.textContent = "Sem dados";
  detailMetaEl.textContent = "A página nao encontrou concursos para desenhar os graficos.";
  chartConfigs.forEach((config) => {
    config.noteEl.textContent = "Sem dados carregados.";
  });
  [
    highlightNumberEl,
    highlightRangeEl,
    opacityRangeEl,
    toggleAllLinesButton,
    rangeStartEl,
    rangeHighlightEl,
    rangeEndEl,
    dateStartEl,
    dateEndEl,
    backgroundToneRangeEl,
    backgroundToneSwatchEl,
    groupFilterEnabledEl,
    groupFilterSourceEl,
    groupFilterSizeEl,
    groupFilterModeEl,
    groupFilterKeyEl,
    recentOverlayLimitEl,
    recentMonthStartEl,
    recentMonthEndEl,
    addSimulationButton,
    autoSimulationButton,
    clearSimulationsButton,
    updateMegaSenaResultButton,
    sorteGameRegisterButton,
    sorteGameResetButton,
    exportPngButton,
    ...simulationInputs,
  ].forEach((element) => {
    element.disabled = true;
  });
};

const updateBackgroundToneControl = () => {
  const background = getOverlayBackgroundStart(state.overlayGradientStart);
  backgroundToneRangeEl.value = String(state.overlayBackgroundTone);
  backgroundToneRangeEl.disabled = !state.showAllLines;
  backgroundToneSwatchEl.value = state.overlayGradientStart;
  backgroundToneSwatchEl.disabled = !state.showAllLines;
  backgroundToneLabelEl.textContent = `${state.overlayGradientStart.toUpperCase()} -> ${describeOverlayBackground()}`;
  backgroundToneSwatchEl.style.background = background;
};

const ensureHighlightVisible = () => {
  state.highlightIndex = clamp(
    state.highlightIndex,
    state.rangeStartIndex,
    state.rangeEndIndex,
  );
};

const getSelectedDraws = () =>
  megaSenaDraws.slice(state.rangeStartIndex, state.rangeEndIndex + 1);

const getCurrentGroupOptions = () =>
  repeatedGroupCatalog.get(
    `${state.groupFilterSource}:${state.groupFilterSize}:${state.groupFilterMode}`,
  ) ?? [];

const describeGroupFilter = () =>
  `${state.groupFilterSize} dezenas | ${groupSourceLabels[state.groupFilterSource]} | ${groupModeLabels[state.groupFilterMode]} | grupo ${state.groupFilterKey || "-"}`;

const describeFilteredDatasetForChart = (config) =>
  config.key === "sorted"
    ? "Mesmo conjunto filtrado, desenhado na ordem crescente do grafico."
    : "Mesmo conjunto filtrado, desenhado na ordem real do sorteio quando disponivel.";

const matchesActiveGroupFilter = (draw) => {
  if (!state.groupFilterEnabled) {
    return true;
  }

  if (!state.groupFilterKey) {
    return false;
  }

  return (
    getGroupKeyForDraw(
      draw,
      state.groupFilterSource,
      state.groupFilterSize,
      state.groupFilterMode,
    ) === state.groupFilterKey
  );
};

const hasActiveMonthOverlayRange = () =>
  Boolean(state.recentMonthStart && state.recentMonthEnd);

const hasActiveDrawIntervalRange = () =>
  state.rangeStartIndex > 0 || state.rangeEndIndex < megaSenaDraws.length - 1;

const getNormalizedMonthOverlayRange = () => {
  if (!hasActiveMonthOverlayRange()) {
    return null;
  }

  return state.recentMonthStart <= state.recentMonthEnd
    ? { start: state.recentMonthStart, end: state.recentMonthEnd }
    : { start: state.recentMonthEnd, end: state.recentMonthStart };
};

const matchesMonthOverlayRange = (draw) => {
  const range = getNormalizedMonthOverlayRange();
  if (!range) {
    return true;
  }

  const month = getMonthKey(draw);
  return month >= range.start && month <= range.end;
};

const isFilteredOverlayModeActive = () =>
  state.showAllLines &&
  (hasActiveMonthOverlayRange() ||
    hasActiveDrawIntervalRange() ||
    state.recentOverlayLimit > 0 ||
    state.groupFilterEnabled);

const getOverlayDraws = (config) =>
  getSelectedDraws()
    .filter(
      (draw) =>
        Array.isArray(config.primaryValuesFor(draw)) &&
        matchesActiveGroupFilter(draw) &&
        matchesMonthOverlayRange(draw),
    )
    .slice(
      !hasActiveMonthOverlayRange() && state.recentOverlayLimit > 0
        ? -state.recentOverlayLimit
        : undefined,
    );

const getSimulationSourceDraws = () => {
  const sortedConfig = chartConfigs.find((config) => config.key === "sorted");
  if (!sortedConfig) {
    return [];
  }

  const filteredDraws = getOverlayDraws(sortedConfig);
  const highlighted = megaSenaDraws[state.highlightIndex];

  if (state.showAllLines) {
    return filteredDraws;
  }

  return filteredDraws.length ? filteredDraws : highlighted ? [highlighted] : [];
};

const getSimulationOptionsByAxis = () => {
  const sourceDraws = getSimulationSourceDraws();
  return sortedAxisLabels.map((_, axisIndex) => {
    const axisValues = sourceDraws
      .map((draw) => draw.dezenas?.[axisIndex])
      .filter((value) => Number.isInteger(value));

    if (!axisValues.length) {
      return [];
    }

    const min = Math.min(...axisValues);
    const max = Math.max(...axisValues);
    return Array.from({ length: max - min + 1 }, (_, index) => min + index);
  });
};

const getSimulationKey = (values) => values.map(formatNumber).join("-");
const getSimulationValues = (game) => (Array.isArray(game) ? game : game.values);

const getCurrentSimulationValues = () =>
  simulationInputs
    .map((input) => (input.value ? Number(input.value) : null))
    .filter(Number.isInteger);

const renderSimulationList = () => {
  if (!state.simulatedGames.length) {
    simulationListEl.innerHTML =
      '<p class="simulation-meta">Nenhum jogo simulado inserido.</p>';
    return;
  }

  simulationListEl.innerHTML = state.simulatedGames
    .map(
      (game, index) => `
        <article class="simulation-item${game.type === "auto" ? " is-auto" : ""}">
          <span>${game.type === "auto" ? "Automático" : "Simulação"} ${index + 1}</span>
          <strong>${formatSequence(getSimulationValues(game))}</strong>
        </article>
      `,
    )
    .join("");
};

const updateSimulationControls = () => {
  const optionsByAxis = getSimulationOptionsByAxis();
  const totalAvailableLines = getSimulationSourceDraws().length;

  simulationInputs.forEach((input, axisIndex) => {
    const previousValue = input.value;
    const options = optionsByAxis[axisIndex] ?? [];
    input.innerHTML = [
      '<option value="">Selecione</option>',
      ...options.map(
        (value) => `<option value="${value}">${formatNumber(value)}</option>`,
      ),
    ].join("");

    if (options.includes(Number(previousValue))) {
      input.value = previousValue;
    } else {
      input.value = "";
    }

    input.disabled = !options.length;
  });

  const currentValues = getCurrentSimulationValues();
  const hasCompleteGame = currentValues.length === sortedAxisLabels.length;
  const hasUniqueNumbers = new Set(currentValues).size === currentValues.length;
  const hasAscendingNumbers = currentValues.every(
    (value, index) => index === 0 || value > currentValues[index - 1],
  );
  const key = hasCompleteGame ? getSimulationKey(currentValues) : "";
  const isDuplicate = state.simulatedGames.some(
    (game) => getSimulationKey(getSimulationValues(game)) === key,
  );

  addSimulationButton.disabled =
    state.isAutoChoosing ||
    !hasCompleteGame ||
    !hasUniqueNumbers ||
    !hasAscendingNumbers ||
    isDuplicate;
  autoSimulationButton.disabled = state.isAutoChoosing || !totalAvailableLines;
  autoSimulationButton.textContent = state.isAutoChoosing
    ? "Escolhendo..."
    : "Escolher automático";
  clearSimulationsButton.disabled =
    state.isAutoChoosing || !state.simulatedGames.length;
  simulationInputs.forEach((input) => {
    input.disabled = state.isAutoChoosing || input.disabled;
  });

  if (state.isAutoChoosing) {
    simulationMetaEl.textContent =
      "Escolha automatica em andamento: cada dezena sera alternada por 10 segundos.";
  } else if (!totalAvailableLines) {
    simulationMetaEl.textContent =
      "Nenhuma linha do primeiro grafico atende aos filtros atuais.";
  } else if (isDuplicate) {
    simulationMetaEl.textContent = "Esse jogo ja foi inserido na simulacao.";
  } else if (hasCompleteGame && (!hasUniqueNumbers || !hasAscendingNumbers)) {
    simulationMetaEl.textContent =
      "Use seis dezenas distintas em ordem crescente para simular no primeiro grafico.";
  } else {
    simulationMetaEl.textContent =
      `${totalAvailableLines} linhas do primeiro grafico definem o menor e maior valor de cada eixo.`;
  }

  renderSimulationList();
};

const addSimulatedGame = (values, type = "manual") => {
  const key = getSimulationKey(values);
  const exists = state.simulatedGames.some(
    (game) => getSimulationKey(getSimulationValues(game)) === key,
  );
  const hasUniqueNumbers = new Set(values).size === values.length;
  const hasAscendingNumbers = values.every(
    (value, index) => index === 0 || value > values[index - 1],
  );

  if (
    values.length !== sortedAxisLabels.length ||
    !hasUniqueNumbers ||
    !hasAscendingNumbers ||
    exists
  ) {
    updateSimulationControls();
    return false;
  }

  state.simulatedGames.push({ values, type });
  simulationInputs.forEach((input) => {
    input.value = "";
  });
  updateSimulationControls();
  renderCharts();
  return true;
};

const sleep = (milliseconds) =>
  new Promise((resolve) => {
    window.setTimeout(resolve, milliseconds);
  });

const buildAutomaticCandidate = (seed) => {
  const optionsByAxis = getSimulationOptionsByAxis();
  const values = [];

  for (let axisIndex = 0; axisIndex < simulationInputs.length; axisIndex += 1) {
    const options = (optionsByAxis[axisIndex] ?? []).filter(
      (value) => !values.includes(value),
    );

    if (!options.length) {
      return null;
    }

    const optionIndex = (seed + axisIndex * 7 + values.length * 3) % options.length;
    values.push(options[optionIndex]);
  }

  return values;
};

const resolveNextAutomaticCandidate = () => {
  const existingKeys = new Set(
    state.simulatedGames.map((game) => getSimulationKey(getSimulationValues(game))),
  );
  const maxAttempts = 1200;

  for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
    const seed = state.autoChoiceRun + attempt;
    const values = buildAutomaticCandidate(seed);
    if (!values) {
      return null;
    }

    const sortedValues = [...values].sort((a, b) => a - b);
    if (!existingKeys.has(getSimulationKey(sortedValues))) {
      state.autoChoiceRun = seed + 1;
      return { values, sortedValues };
    }
  }

  return null;
};

const chooseAutomaticSimulation = async () => {
  if (state.isAutoChoosing) {
    return;
  }

  const candidate = resolveNextAutomaticCandidate();
  if (!candidate) {
    simulationMetaEl.textContent =
      "Nao ha nova combinacao automatica disponivel para os filtros atuais.";
    updateSimulationControls();
    return;
  }

  state.isAutoChoosing = true;
  const selectedValues = [];
  updateSimulationControls();

  for (let axisIndex = 0; axisIndex < simulationInputs.length; axisIndex += 1) {
    const input = simulationInputs[axisIndex];
    const baseOptions = getSimulationOptionsByAxis()[axisIndex] ?? [];
    const options = baseOptions.filter((value) => !selectedValues.includes(value));

    if (!options.length) {
      state.isAutoChoosing = false;
      simulationMetaEl.textContent =
        "Nao ha dezenas disponiveis para concluir a escolha automatica com os filtros atuais.";
      updateSimulationControls();
      return;
    }

    const targetValue = candidate.values[axisIndex];
    input.disabled = false;
    input.innerHTML = options
      .map((value) => `<option value="${value}">${formatNumber(value)}</option>`)
      .join("");

    let optionIndex = Math.max(0, options.indexOf(targetValue));
    input.value = String(options[optionIndex]);
    const intervalId = window.setInterval(() => {
      optionIndex = (optionIndex + 1) % options.length;
      input.value = String(options[optionIndex]);
    }, 137);

    await sleep(10000);
    window.clearInterval(intervalId);
    input.value = String(targetValue);
    const selectedValue = Number(input.value);
    selectedValues.push(selectedValue);
    input.disabled = true;
  }

  state.isAutoChoosing = false;
  const added = addSimulatedGame([...selectedValues].sort((a, b) => a - b), "auto");
  if (!added) {
    simulationMetaEl.textContent =
      "A escolha automatica gerou um jogo ja existente ou invalido para os filtros atuais.";
  }
};

const updateRecentOverlayControls = () => {
  const monthRange = getNormalizedMonthOverlayRange();
  const monthRangeCount = monthRange
    ? getSelectedDraws().filter(
        (draw) =>
          getMonthKey(draw) >= monthRange.start && getMonthKey(draw) <= monthRange.end,
      ).length
    : 0;

  recentOverlayLimitEl.value = String(state.recentOverlayLimit);
  recentOverlayLimitEl.disabled = !state.showAllLines;
  recentMonthStartEl.value = state.recentMonthStart;
  recentMonthEndEl.value = state.recentMonthEnd;
  recentMonthStartEl.disabled = !state.showAllLines;
  recentMonthEndEl.disabled = !state.showAllLines;

  if (!state.showAllLines) {
    recentOverlayMetaEl.textContent =
      "Disponivel quando o modo Mostrar todas as linhas estiver ativo.";
    return;
  }

  if (monthRange) {
    recentOverlayMetaEl.textContent =
      `Periodo ativo: ${monthRange.start} a ${monthRange.end}, com ${monthRangeCount} concursos ` +
      "no recorte atual. A quantidade de concursos recentes foi ignorada.";
    return;
  }

  if (state.recentOverlayLimit > 0) {
    recentOverlayMetaEl.textContent =
      `As linhas de fundo vao considerar ate os ${state.recentOverlayLimit} concursos mais recentes ` +
      "do recorte atual, cada um com uma cor exclusiva sem usar azul ou vermelho.";
    return;
  }

  recentOverlayMetaEl.textContent =
    "Quando ativado, esse recorte reduz as linhas de fundo para os concursos mais recentes do intervalo atual, em passos de 10 ate 50.";
};

const updateGroupFilterControls = () => {
  const options = getCurrentGroupOptions();
  const selectedOption =
    options.find((item) => item.key === state.groupFilterKey) ?? options[0] ?? null;

  state.groupFilterKey = selectedOption?.key ?? "";

  groupFilterEnabledEl.checked = state.groupFilterEnabled;
  groupFilterSourceEl.value = state.groupFilterSource;
  groupFilterSizeEl.value = String(state.groupFilterSize);
  groupFilterModeEl.value = state.groupFilterMode;

  groupFilterKeyEl.innerHTML = options.length
    ? options
        .map(
          (item) =>
            `<option value="${item.key}">${item.key} | ${item.count} ocorrencias</option>`,
        )
        .join("")
    : `<option value="">Nenhum grupo repetido</option>`;

  if (state.groupFilterKey) {
    groupFilterKeyEl.value = state.groupFilterKey;
  }

  const controlsDisabled = !state.showAllLines;
  groupFilterEnabledEl.disabled = controlsDisabled;
  groupFilterSourceEl.disabled = controlsDisabled;
  groupFilterSizeEl.disabled = controlsDisabled;
  groupFilterModeEl.disabled = controlsDisabled;
  groupFilterKeyEl.disabled =
    controlsDisabled || !state.groupFilterEnabled || !options.length;

  if (controlsDisabled) {
    groupFilterMetaEl.textContent =
      "Disponivel quando o modo Mostrar todas as linhas estiver ativo.";
    return;
  }

  if (!options.length) {
    groupFilterMetaEl.textContent =
      "Nenhum grupo repetido foi encontrado para essa combinacao de base, tamanho e modo.";
    return;
  }

  if (state.groupFilterEnabled && selectedOption) {
    groupFilterMetaEl.textContent =
      `Filtro ativo para ${describeGroupFilter()}. ` +
      `Esse grupo aparece ${selectedOption.count} vezes no historico carregado.`;
    return;
  }

  groupFilterMetaEl.textContent =
    `${options.length} grupos repetidos foram encontrados para ` +
    `${state.groupFilterSize} dezenas em ${groupSourceLabels[state.groupFilterSource]} ` +
    `(${groupModeLabels[state.groupFilterMode]}).`;
};

const updateRangePanel = () => {
  const total = megaSenaDraws.length;
  const maxIndex = Math.max(0, total - 1);
  const startDraw = megaSenaDraws[state.rangeStartIndex];
  const endDraw = megaSenaDraws[state.rangeEndIndex];
  const selectedCount = state.rangeEndIndex - state.rangeStartIndex + 1;
  const startPct = maxIndex === 0 ? 0 : (state.rangeStartIndex / maxIndex) * 100;
  const endPct = maxIndex === 0 ? 100 : (state.rangeEndIndex / maxIndex) * 100;

  rangePanelEl.hidden = !state.showAllLines;
  opacityRangeEl.disabled = !state.showAllLines;
  rangeStartEl.disabled = !state.showAllLines;
  rangeHighlightEl.disabled = !state.showAllLines;
  rangeEndEl.disabled = !state.showAllLines;
  dateStartEl.disabled = !state.showAllLines;
  dateEndEl.disabled = !state.showAllLines;
  updateBackgroundToneControl();
  updateGroupFilterControls();
  updateRecentOverlayControls();
  updateSimulationControls();

  if (!startDraw || !endDraw) {
    return;
  }

  rangeCountEl.textContent = `${selectedCount} de ${total} concursos no intervalo`;
  rangeStartLabelEl.innerHTML = `<span>Inicio</span><strong>#${startDraw.concurso} | ${formatDate(startDraw.dataSorteio)}</strong>`;
  rangeEndLabelEl.innerHTML = `<span>Fim</span><strong>#${endDraw.concurso} | ${formatDate(endDraw.dataSorteio)}</strong>`;
  rangeStartEl.value = String(state.rangeStartIndex);
  rangeHighlightEl.value = String(state.highlightIndex);
  rangeEndEl.value = String(state.rangeEndIndex);
  dateStartEl.value = getDateKey(startDraw);
  dateEndEl.value = getDateKey(endDraw);
  rangeFillEl.style.left = `${startPct}%`;
  rangeFillEl.style.width = `${Math.max(0, endPct - startPct)}%`;
};

const buildDetailCards = (labels, values, unavailableMessage) => {
  if (!Array.isArray(values)) {
    return `<article class="detail-line"><span>Disponibilidade</span><strong>${unavailableMessage}</strong></article>`;
  }

  return labels
    .map(
      (label, index) => `
        <article class="detail-line">
          <span>${label}</span>
          <strong>${formatNumber(values[index])}</strong>
        </article>
      `,
    )
    .join("");
};

const updateDetailPanel = () => {
  const draw = megaSenaDraws[state.highlightIndex];
  if (!draw) {
    return;
  }

  detailTitleEl.textContent = `Concurso ${draw.concurso}`;
  detailMetaEl.textContent = `Sorteio em ${formatDate(draw.dataSorteio)}`;
  detailGridSortedEl.innerHTML = buildDetailCards(
    sortedAxisLabels,
    draw.dezenas,
    "Sem dezenas para exibir.",
  );
  detailGridOrderedEl.innerHTML = buildDetailCards(
    orderedAxisLabels,
    draw.ordemSorteio,
    "Ordem indisponivel na base para este concurso.",
  );
  detailOrderMetaEl.textContent = draw.ordemSorteio
    ? "A linha do segundo grafico usa a sequencia real de saida das dezenas."
    : "A ordem do sorteio foi carregada apenas do concurso 2196 em diante.";

  highlightNumberEl.value = String(draw.concurso);
  highlightRangeEl.value = String(draw.concurso);
  updateSimulationControls();
};

const clearSorteGameTimers = () => {
  sorteGameState.timers.forEach((timerId) => window.clearTimeout(timerId));
  sorteGameState.timers = [];
};

const renderSorteGame = () => {
  if (!sorteGameBoardEl) {
    return;
  }

  sorteGameBoardEl.innerHTML = sorteGameState.slots
    .map((slot, index) => {
      const isCurrent = index === sorteGameState.nextIndex;
      const isDone = Number.isInteger(slot.value) && Boolean(slot.drawnAt);
      const buttonLabel =
        sorteGameState.animatingIndex === index
          ? "Sorteando..."
          : isDone
            ? "Definido"
            : isCurrent
              ? `Sortear ${index + 1}`
              : `Aguardar ${index + 1}`;
      return `
        <article class="sorte-slot">
          <div class="sorte-ball">${slot.value ? formatNumber(slot.value) : "--"}</div>
          <button class="secondary-button sorte-draw-button" type="button" data-sorte-index="${index}" ${
            !isCurrent || sorteGameState.animatingIndex !== null || isDone ? "disabled" : ""
          }>${buttonLabel}</button>
        </article>
      `;
    })
    .join("");

  sorteGameBoardEl.querySelectorAll(".sorte-draw-button").forEach((button) => {
    button.addEventListener("click", () => {
      drawSorteGameSlot(Number(button.dataset.sorteIndex));
    });
  });

  const finished = sorteGameState.nextIndex >= sorteGameState.slots.length;
  sorteGameRegisterButton.disabled =
    !finished || sorteGameState.registerPending || Boolean(sorteGameState.savedSequence);
  sorteGameResetButton.disabled = sorteGameState.animatingIndex !== null;
};

const resetSorteGame = () => {
  clearSorteGameTimers();
  sorteGameState.slots = Array.from({ length: 6 }, () => ({ value: null, drawnAt: null }));
  sorteGameState.nextIndex = 0;
  sorteGameState.animatingIndex = null;
  sorteGameState.savedSequence = null;
  sorteGameState.registerPending = false;
  sorteGameStatusEl.textContent = "Acione o primeiro botão para iniciar a sequência.";
  sorteGameRegisterStatusEl.textContent = "-";
  renderSorteGame();
};

const drawSorteGameSlot = (index) => {
  if (index !== sorteGameState.nextIndex || sorteGameState.animatingIndex !== null) {
    return;
  }

  clearSorteGameTimers();
  sorteGameRegisterStatusEl.textContent = "-";
  const selected = new Set(
    sorteGameState.slots
      .map((slot) => slot.value)
      .filter((value) => Number.isInteger(value)),
  );
  const available = Array.from({ length: 60 }, (_, valueIndex) => valueIndex + 1).filter(
    (value) => !selected.has(value),
  );
  if (!available.length) {
    return;
  }

  sorteGameState.animatingIndex = index;
  sorteGameStatusEl.textContent = `Sorteando a dezena ${index + 1}.`;
  renderSorteGame();

  let cursor = 0;
  const frameMs = Math.max(80, Math.floor(5000 / available.length));
  available.forEach((_, sequenceIndex) => {
    const timerId = window.setTimeout(() => {
      cursor = (cursor + 1) % available.length;
      sorteGameState.slots[index] = { value: available[cursor], drawnAt: null };
      renderSorteGame();
    }, sequenceIndex * frameMs);
    sorteGameState.timers.push(timerId);
  });

  const finalValue = available[Math.floor(Math.random() * available.length)];
  const finalizeTimerId = window.setTimeout(() => {
    const confirmedAt = new Date().toISOString();
    sorteGameState.slots[index] = { value: finalValue, drawnAt: confirmedAt };
    sorteGameState.animatingIndex = null;
    sorteGameState.nextIndex = index + 1;
    sorteGameStatusEl.textContent =
      index === sorteGameState.slots.length - 1
        ? "Simulação encerrada. As seis dezenas foram definidas sem repetição."
        : `Dezena ${formatNumber(finalValue)} definida. Acione o próximo botão.`;
    sorteGameState.timers = [];
    renderSorteGame();
  }, 5000);
  sorteGameState.timers.push(finalizeTimerId);
};

const buildSorteGamePayload = () => ({
  dezenas: sorteGameState.slots.map((slot, index) => {
    const drawnAt = new Date(slot.drawnAt);
    return {
      posicao: index + 1,
      dezena: slot.value,
      sorteadoEm: slot.drawnAt,
      data: `${drawnAt.getFullYear()}-${String(drawnAt.getMonth() + 1).padStart(2, "0")}-${String(drawnAt.getDate()).padStart(2, "0")}`,
      hora: drawnAt.getHours(),
      minuto: drawnAt.getMinutes(),
      segundo: drawnAt.getSeconds(),
    };
  }),
});

const registerSorteGame = async () => {
  const finished = sorteGameState.nextIndex >= sorteGameState.slots.length;
  if (!finished || sorteGameState.registerPending || sorteGameState.savedSequence) {
    return;
  }

  sorteGameState.registerPending = true;
  sorteGameRegisterStatusEl.textContent = "Registrando simulação no backend da Mega-Sena...";
  renderSorteGame();

  try {
    const response = await fetch(MEGA_SENA_SIMULATION_ENDPOINT, {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(buildSorteGamePayload()),
    });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(payload.message || "Nao foi possivel registrar a simulacao.");
    }
    sorteGameState.savedSequence = payload.sequencial;
    sorteGameRegisterStatusEl.textContent = `Simulação registrada com sequencial ${payload.sequencial}.`;
  } catch (error) {
    sorteGameRegisterStatusEl.textContent =
      error instanceof Error ? error.message : "Nao foi possivel registrar a simulacao.";
  } finally {
    sorteGameState.registerPending = false;
    renderSorteGame();
  }
};

const upsertLocalMegaSenaDraw = (result) => {
  const concurso = Number(result.concurso);
  const dezenas = Array.isArray(result.dezenas) ? result.dezenas.map(Number) : [];
  const ordemSorteio = Array.isArray(result.ordemSorteio)
    ? result.ordemSorteio.map(Number)
    : null;
  const dataSorteio = result.dataSorteio
    ? `${result.dataSorteio}T00:00:00.000Z`
    : null;

  if (
    !Number.isInteger(concurso) ||
    dezenas.length !== 6 ||
    !ordemSorteio ||
    ordemSorteio.length !== 6 ||
    !dataSorteio
  ) {
    return;
  }

  const nextDraw = { concurso, dataSorteio, dezenas, ordemSorteio };
  const existingIndex = drawMap.get(concurso);

  if (Number.isInteger(existingIndex)) {
    megaSenaDraws[existingIndex] = nextDraw;
  } else {
    megaSenaDraws.push(nextDraw);
    megaSenaDraws.sort((left, right) => left.concurso - right.concurso);
    drawMap.clear();
    megaSenaDraws.forEach((draw, index) => drawMap.set(draw.concurso, index));
    state.rangeEndIndex = megaSenaDraws.length - 1;
  }

  state.highlightIndex = drawMap.get(concurso) ?? state.highlightIndex;
  setupStats();
  updateDetailPanel();
  updateRangePanel();
  renderCharts();
};

const includeHighlightInRange = (highlightIndex) => {
  if (highlightIndex < state.rangeStartIndex) {
    state.rangeStartIndex = highlightIndex;
  }
  if (highlightIndex > state.rangeEndIndex) {
    state.rangeEndIndex = highlightIndex;
  }
};

const updateAllLinesButton = () => {
  toggleAllLinesButton.textContent = state.showAllLines
    ? "Ocultar todas as linhas"
    : "Mostrar todas as linhas";
  toggleAllLinesButton.setAttribute("aria-pressed", String(state.showAllLines));
  updateRangePanel();
};

const findStartIndexByDate = (dateKey) => {
  const nextIndex = megaSenaDraws.findIndex((draw) => getDateKey(draw) >= dateKey);
  return nextIndex === -1 ? megaSenaDraws.length - 1 : nextIndex;
};

const findEndIndexByDate = (dateKey) => {
  for (let index = megaSenaDraws.length - 1; index >= 0; index -= 1) {
    if (getDateKey(megaSenaDraws[index]) <= dateKey) {
      return index;
    }
  }
  return 0;
};

const applyRange = (startIndex, endIndex) => {
  state.rangeStartIndex = clamp(startIndex, 0, megaSenaDraws.length - 1);
  state.rangeEndIndex = clamp(
    endIndex,
    state.rangeStartIndex,
    megaSenaDraws.length - 1,
  );
  ensureHighlightVisible();
  updateDetailPanel();
  updateRangePanel();
  renderCharts();
};

const setupStats = () => {
  const first = megaSenaDraws[0];
  const last = megaSenaDraws[megaSenaDraws.length - 1];

  totalDrawsEl.textContent = String(megaSenaDraws.length);
  firstDrawEl.textContent = `#${first.concurso}`;
  lastDrawEl.textContent = `#${last.concurso}`;
  dateRangeEl.textContent = `${formatDate(first.dataSorteio)} a ${formatDate(last.dataSorteio)}`;

  highlightNumberEl.min = String(first.concurso);
  highlightNumberEl.max = String(last.concurso);
  highlightRangeEl.min = String(first.concurso);
  highlightRangeEl.max = String(last.concurso);
  rangeStartEl.min = "0";
  rangeStartEl.max = String(megaSenaDraws.length - 1);
  rangeHighlightEl.min = "0";
  rangeHighlightEl.max = String(megaSenaDraws.length - 1);
  rangeEndEl.min = "0";
  rangeEndEl.max = String(megaSenaDraws.length - 1);
  dateStartEl.min = getDateKey(first);
  dateStartEl.max = getDateKey(last);
  dateEndEl.min = getDateKey(first);
  dateEndEl.max = getDateKey(last);
  recentMonthStartEl.min = getMonthKey(first);
  recentMonthStartEl.max = getMonthKey(last);
  recentMonthEndEl.min = getMonthKey(first);
  recentMonthEndEl.max = getMonthKey(last);
};

const resizeCanvases = () => {
  const dpr = window.devicePixelRatio || 1;

  chartConfigs.forEach((config) => {
    const rect = config.shellEl.getBoundingClientRect();
    config.canvasEl.width = Math.round(rect.width * dpr);
    config.canvasEl.height = Math.round(rect.height * dpr);
    config.ctx.setTransform(1, 0, 0, 1, 0, 0);
    config.ctx.scale(dpr, dpr);
  });

  renderCharts();
};

const getPalette = () => {
  if (state.showAllLines) {
    return {
      backgroundStart: getOverlayBackgroundStart(),
      backgroundEnd: getOverlayBackgroundEnd(),
      gridMajor: "rgba(15, 23, 36, 0.14)",
      gridMinor: "rgba(15, 23, 36, 0.05)",
      axis: "rgba(15, 23, 36, 0.48)",
      axisTick: "rgba(15, 23, 36, 0.24)",
      label: "#18324a",
      tickText: "#556b82",
      highlight: "rgba(13, 110, 253, 0.92)",
      highlightPoint: "rgba(13, 110, 253, 1)",
    };
  }

  return {
    background: "#0b1220",
    gridMajor: "rgba(113, 139, 176, 0.26)",
    gridMinor: "rgba(113, 139, 176, 0.08)",
    axis: "rgba(208, 224, 255, 0.52)",
    axisTick: "rgba(208, 224, 255, 0.14)",
    label: "#9ed6ff",
    tickText: "#7f93aa",
    highlight: "rgba(79, 179, 255, 0.96)",
    highlightPoint: "rgba(158, 214, 255, 1)",
  };
};

const addRoundedRectPath = (ctx, x, y, width, height, radius) => {
  const safeRadius = Math.min(radius, width / 2, height / 2);
  ctx.beginPath();
  ctx.moveTo(x + safeRadius, y);
  ctx.lineTo(x + width - safeRadius, y);
  ctx.quadraticCurveTo(x + width, y, x + width, y + safeRadius);
  ctx.lineTo(x + width, y + height - safeRadius);
  ctx.quadraticCurveTo(x + width, y + height, x + width - safeRadius, y + height);
  ctx.lineTo(x + safeRadius, y + height);
  ctx.quadraticCurveTo(x, y + height, x, y + height - safeRadius);
  ctx.lineTo(x, y + safeRadius);
  ctx.quadraticCurveTo(x, y, x + safeRadius, y);
  ctx.closePath();
};

const drawEmptyState = (ctx, width, height, palette, message) => {
  if (!message) {
    return;
  }

  const boxWidth = Math.min(620, width - 80);
  const boxHeight = 92;
  const x = (width - boxWidth) / 2;
  const y = (height - boxHeight) / 2;

  ctx.save();
  ctx.fillStyle = "rgba(15, 23, 36, 0.10)";
  ctx.strokeStyle = "rgba(15, 23, 36, 0.16)";
  ctx.lineWidth = 1.5;
  addRoundedRectPath(ctx, x, y, boxWidth, boxHeight, 18);
  ctx.fill();
  ctx.stroke();
  ctx.fillStyle = palette.label;
  ctx.font = "600 15px Space Grotesk, Segoe UI, sans-serif";
  ctx.textAlign = "center";
  ctx.fillText(message, width / 2, y + 42);
  ctx.font = "12px Space Grotesk, Segoe UI, sans-serif";
  ctx.fillStyle = palette.tickText;
  ctx.fillText(
    "Os filtros continuam compartilhados com o grafico principal.",
    width / 2,
    y + 66,
  );
  ctx.restore();
};

const buildChartNote = (
  config,
  highlighted,
  overlayLineCount,
  selectedCount,
  overlayUniverseCount,
) => {
  const primaryValues = config.primaryValuesFor(highlighted);
  const comparisonValues = config.comparisonValuesFor(highlighted);
  const showComparisonSeries = !isFilteredOverlayModeActive();

  const parts = [
    primaryValues
      ? `${config.primarySeries.label}: ${formatSequence(primaryValues)}`
      : `${config.primarySeries.label}: ${config.primarySeries.missingLabel}`,
  ];

  if (showComparisonSeries) {
    parts.push(
      comparisonValues
      ? `${config.comparisonSeries.label}: ${formatSequence(comparisonValues)}`
      : `${config.comparisonSeries.label}: ${config.comparisonSeries.missingLabel}`,
    );
  }

  if (state.showAllLines) {
    const monthRange = getNormalizedMonthOverlayRange();
    const intervalSummary = monthRange
      ? `${overlayLineCount} linhas de fundo ativas no periodo ${monthRange.start} a ${monthRange.end}`
      : state.groupFilterEnabled
      ? `${overlayLineCount} linhas de fundo ativas para ${describeGroupFilter()}`
      : config.key === "ordered"
        ? `${overlayLineCount} de ${selectedCount} linhas de fundo ativas no intervalo`
        : `${overlayLineCount} linhas de fundo ativas no intervalo`;
    const recentSummary =
      !monthRange && state.recentOverlayLimit > 0
        ? ` | recorte colorido: ${overlayUniverseCount} concursos elegiveis entre os ultimos ${state.recentOverlayLimit}`
        : "";
    const datasetSummary = state.groupFilterEnabled || monthRange
      ? ` | ${describeFilteredDatasetForChart(config)}`
      : "";
    return `Concurso ${highlighted.concurso} | ${intervalSummary}${recentSummary}${datasetSummary} | ${parts.join(" | ")}`;
  }

  return `Concurso ${highlighted.concurso} | ${parts.join(" | ")}`;
};

const getSeriesAppearance = (seriesColor, palette) => {
  if (seriesColor === "red") {
    return state.showAllLines
      ? {
          stroke: "rgba(208, 46, 70, 0.94)",
          point: "rgba(208, 46, 70, 1)",
        }
      : {
          stroke: "rgba(255, 112, 112, 0.96)",
          point: "rgba(255, 168, 168, 1)",
        };
  }

  return {
    stroke: palette.highlight,
    point: palette.highlightPoint,
  };
};

const renderChart = (config) => {
  const ctx = config.ctx;
  const width = config.canvasEl.clientWidth;
  const height = config.canvasEl.clientHeight;
  const margin = { top: 72, right: 64, bottom: 56, left: 64 };
  const plotWidth = width - margin.left - margin.right;
  const plotHeight = height - margin.top - margin.bottom;
  const palette = getPalette();
  const tickValues = Array.from({ length: 60 }, (_, index) => index + 1);
  const highlighted = megaSenaDraws[state.highlightIndex];
  const selectedCount = state.rangeEndIndex - state.rangeStartIndex + 1;
  const overlayDraws = getOverlayDraws(config);
  const primaryValues = config.primaryValuesFor(highlighted);
  const comparisonValues = config.comparisonValuesFor(highlighted);
  const primaryAppearance = getSeriesAppearance(config.primarySeries.color, palette);
  const comparisonAppearance = getSeriesAppearance(
    config.comparisonSeries.color,
    palette,
  );
  const highlightLineWidth = 3.2;
  const comparisonLineWidth = 2.4;
  const backgroundLineWidth = Math.max(highlightLineWidth / 4, 1.4);
  const overlayBackgroundDraws = overlayDraws.filter(
    (draw) => !(primaryValues && draw.concurso === highlighted.concurso),
  );
  const useColoredOverlay =
    state.showAllLines &&
    (state.recentOverlayLimit > 0 || hasActiveMonthOverlayRange());
  const showComparisonSeries = !isFilteredOverlayModeActive();
  const overlayStrokeStyle = state.groupFilterEnabled
    ? "rgba(17, 24, 39, 0.72)"
    : `rgba(0, 0, 0, ${Math.max(0.16, state.baseOpacity)})`;
  const overlayPointFill = state.groupFilterEnabled
    ? "rgba(17, 24, 39, 0.88)"
    : `rgba(0, 0, 0, ${Math.max(0.32, state.baseOpacity + 0.1)})`;
  const overlayLineWidth = useColoredOverlay
    ? Math.max(backgroundLineWidth, 2.05)
    : state.groupFilterEnabled
    ? Math.max(backgroundLineWidth, 2.2)
    : backgroundLineWidth;
  const overlayPointRadius = useColoredOverlay
    ? 2.7
    : state.groupFilterEnabled
      ? 2.6
      : 1.9;

  ctx.clearRect(0, 0, width, height);

  if (state.showAllLines) {
    const gradient = ctx.createLinearGradient(0, 0, width, height);
    gradient.addColorStop(0, palette.backgroundStart);
    gradient.addColorStop(1, palette.backgroundEnd);
    ctx.fillStyle = gradient;
  } else {
    ctx.fillStyle = palette.background;
  }
  ctx.fillRect(0, 0, width, height);

  const axisXs = config.axisLabels.map((_, index) => {
    if (config.axisLabels.length === 1) {
      return margin.left + plotWidth / 2;
    }
    return margin.left + (plotWidth * index) / (config.axisLabels.length - 1);
  });

  const valueToY = (value) => {
    const ratio = (60 - value) / 59;
    return margin.top + ratio * plotHeight;
  };

  const drawLine = (values, options) => {
    const points = values.map((value, axisIndex) => ({
      x: axisXs[axisIndex],
      y: valueToY(value),
    }));

    ctx.beginPath();
    points.forEach((point, index) => {
      if (index === 0) {
        ctx.moveTo(point.x, point.y);
      } else {
        ctx.lineTo(point.x, point.y);
      }
    });
    ctx.strokeStyle = options.strokeStyle;
    ctx.lineWidth = options.lineWidth;
    ctx.stroke();

    points.forEach((point) => {
      ctx.beginPath();
      ctx.arc(point.x, point.y, options.pointRadius, 0, Math.PI * 2);
      ctx.fillStyle = options.pointFill;
      ctx.fill();
    });
  };

  tickValues.forEach((tick) => {
    const y = valueToY(tick);
    ctx.strokeStyle =
      tick % 5 === 0 || tick === 1 || tick === 60
        ? palette.gridMajor
        : palette.gridMinor;
    ctx.beginPath();
    ctx.moveTo(margin.left - 6, y);
    ctx.lineTo(width - margin.right + 6, y);
    ctx.stroke();
  });

  axisXs.forEach((x, axisIndex) => {
    ctx.strokeStyle = palette.axis;
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.moveTo(x, margin.top);
    ctx.lineTo(x, height - margin.bottom);
    ctx.stroke();

    ctx.fillStyle = palette.label;
    ctx.font = "600 15px Space Grotesk, Segoe UI, sans-serif";
    ctx.textAlign = "center";
    ctx.fillText(config.axisLabels[axisIndex], x, 34);

    ctx.font = "11px Space Grotesk, Segoe UI, sans-serif";
    tickValues.forEach((tick) => {
      const y = valueToY(tick);
      ctx.strokeStyle =
        tick % 5 === 0 || tick === 1 || tick === 60
          ? palette.axis
          : palette.axisTick;
      ctx.beginPath();
      ctx.moveTo(x - 6, y);
      ctx.lineTo(x + 6, y);
      ctx.stroke();

      if (tick % 5 === 0 || tick === 1 || tick === 60) {
        ctx.fillStyle = palette.tickText;
        ctx.textAlign =
          axisIndex === 0
            ? "right"
            : axisIndex === config.axisLabels.length - 1
              ? "left"
              : "center";
        const labelX =
          axisIndex === 0
            ? x - 10
            : axisIndex === config.axisLabels.length - 1
              ? x + 10
              : x;
        ctx.fillText(String(tick), labelX, y + 4);
      }
    });
  });

  if (state.showAllLines) {
    overlayBackgroundDraws.forEach((draw) => {
      const values = config.primaryValuesFor(draw);
      if (!values) {
        return;
      }

      drawLine(values, {
        strokeStyle: useColoredOverlay
          ? getOverlayLineColor(draw, Math.max(0.62, state.baseOpacity + 0.42))
          : overlayStrokeStyle,
        lineWidth: overlayLineWidth,
        pointRadius: overlayPointRadius,
        pointFill: useColoredOverlay
          ? getOverlayLineColor(draw, 0.94, true)
          : overlayPointFill,
      });
    });
  }

  if (showComparisonSeries && comparisonValues) {
    drawLine(comparisonValues, {
      strokeStyle: comparisonAppearance.stroke,
      lineWidth: comparisonLineWidth,
      pointRadius: 3.2,
      pointFill: comparisonAppearance.point,
    });
  }

  if (primaryValues) {
    drawLine(primaryValues, {
      strokeStyle: primaryAppearance.stroke,
      lineWidth: highlightLineWidth,
      pointRadius: 4.5,
      pointFill: primaryAppearance.point,
    });
  } else if (!showComparisonSeries || !comparisonValues) {
    drawEmptyState(ctx, width, height, palette, config.unavailableMessage);
  }

  if (config.key === "sorted" && state.simulatedGames.length) {
    state.simulatedGames.forEach((game) => {
      const isAuto = game.type === "auto";
      drawLine(getSimulationValues(game), {
        strokeStyle: isAuto
          ? "rgba(255, 59, 59, 0.98)"
          : "rgba(255, 214, 10, 0.96)",
        lineWidth: 3.6,
        pointRadius: 4.2,
        pointFill: isAuto
          ? "rgba(255, 171, 171, 1)"
          : "rgba(255, 236, 153, 1)",
      });
    });
  }

  ctx.fillStyle = palette.label;
  ctx.font = "600 14px Space Grotesk, Segoe UI, sans-serif";
  ctx.textAlign = "left";
  const footerText = state.showAllLines
    ? `Intervalo ativo: ${formatDate(megaSenaDraws[state.rangeStartIndex].dataSorteio)} a ${formatDate(megaSenaDraws[state.rangeEndIndex].dataSorteio)} | Destaque: ${highlighted.concurso}`
    : `Concurso em destaque: ${highlighted.concurso}`;
  ctx.fillText(footerText, margin.left, height - 18);

  config.noteEl.textContent = buildChartNote(
    config,
    highlighted,
    overlayBackgroundDraws.length,
    selectedCount,
    overlayDraws.length,
  );
};

const renderCharts = () => {
  chartConfigs.forEach(renderChart);
};

const setHighlightByIndex = (nextIndex, options = {}) => {
  if (typeof nextIndex !== "number" || Number.isNaN(nextIndex)) {
    return;
  }

  const { expandRange = state.showAllLines } = options;
  state.highlightIndex = nextIndex;

  if (state.showAllLines) {
    if (expandRange) {
      includeHighlightInRange(nextIndex);
    } else {
      ensureHighlightVisible();
    }
    updateRangePanel();
  }

  updateDetailPanel();
  renderCharts();
};

const setHighlightByConcurso = (concurso) => {
  const nextIndex = drawMap.get(Number(concurso));
  setHighlightByIndex(nextIndex);
};

highlightNumberEl.addEventListener("change", (event) => {
  setHighlightByConcurso(event.target.value);
});

highlightRangeEl.addEventListener("input", (event) => {
  setHighlightByConcurso(event.target.value);
});

opacityRangeEl.addEventListener("input", (event) => {
  state.baseOpacity = Number(event.target.value) / 100;
  renderCharts();
});

rangeStartEl.addEventListener("input", (event) => {
  const nextStart = Math.min(Number(event.target.value), state.rangeEndIndex);
  applyRange(nextStart, state.rangeEndIndex);
});

rangeHighlightEl.addEventListener("input", (event) => {
  const nextIndex = clamp(
    Number(event.target.value),
    state.rangeStartIndex,
    state.rangeEndIndex,
  );
  setHighlightByIndex(nextIndex, { expandRange: false });
});

rangeEndEl.addEventListener("input", (event) => {
  const nextEnd = Math.max(Number(event.target.value), state.rangeStartIndex);
  applyRange(state.rangeStartIndex, nextEnd);
});

groupFilterEnabledEl.addEventListener("change", (event) => {
  state.groupFilterEnabled = event.target.checked;
  updateRangePanel();
  renderCharts();
});

groupFilterSourceEl.addEventListener("change", (event) => {
  state.groupFilterSource = event.target.value;
  updateRangePanel();
  renderCharts();
});

groupFilterSizeEl.addEventListener("change", (event) => {
  state.groupFilterSize = Number(event.target.value);
  updateRangePanel();
  renderCharts();
});

groupFilterModeEl.addEventListener("change", (event) => {
  state.groupFilterMode = event.target.value;
  updateRangePanel();
  renderCharts();
});

groupFilterKeyEl.addEventListener("change", (event) => {
  state.groupFilterKey = event.target.value;
  updateGroupFilterControls();
  updateSimulationControls();
  renderCharts();
});

recentOverlayLimitEl.addEventListener("change", (event) => {
  state.recentOverlayLimit = Number(event.target.value);
  updateRecentOverlayControls();
  updateSimulationControls();
  renderCharts();
});

recentMonthStartEl.addEventListener("change", (event) => {
  state.recentMonthStart = event.target.value;
  updateRecentOverlayControls();
  updateSimulationControls();
  renderCharts();
});

recentMonthEndEl.addEventListener("change", (event) => {
  state.recentMonthEnd = event.target.value;
  updateRecentOverlayControls();
  updateSimulationControls();
  renderCharts();
});

simulationInputs.forEach((input) => {
  input.addEventListener("change", () => {
    updateSimulationControls();
  });
});

addSimulationButton.addEventListener("click", () => {
  addSimulatedGame(getCurrentSimulationValues());
});

autoSimulationButton.addEventListener("click", () => {
  chooseAutomaticSimulation();
});

clearSimulationsButton.addEventListener("click", () => {
  state.simulatedGames = [];
  updateSimulationControls();
  renderCharts();
});

updateMegaSenaResultButton.addEventListener("click", async () => {
  updateMegaSenaResultButton.disabled = true;
  updateMegaSenaStatusEl.textContent = "Consultando fonte oficial e atualizando MongoDB...";

  try {
    const response = await fetch(MEGA_SENA_UPDATE_ENDPOINT, {
      method: "POST",
      headers: {
        Accept: "application/json",
      },
    });
    const payload = await response.json().catch(() => ({}));

    if (!response.ok) {
      throw new Error(payload.message || "Falha ao atualizar resultado oficial.");
    }

    upsertLocalMegaSenaDraw(payload);
    updateMegaSenaStatusEl.textContent =
      `Concurso ${payload.concurso} atualizado no MongoDB em ${new Date(payload.atualizadoEm).toLocaleString("pt-BR")}.`;
  } catch (error) {
    updateMegaSenaStatusEl.textContent =
      error instanceof Error ? error.message : "Falha ao atualizar resultado oficial.";
  } finally {
    updateMegaSenaResultButton.disabled = false;
  }
});

sorteGameRegisterButton.addEventListener("click", () => {
  registerSorteGame();
});

sorteGameResetButton.addEventListener("click", () => {
  resetSorteGame();
});

[rangeStartEl, rangeHighlightEl, rangeEndEl].forEach((element) => {
  const activate = () => {
    [rangeStartEl, rangeHighlightEl, rangeEndEl].forEach((input) => {
      input.classList.toggle("is-active", input === element);
    });
  };

  element.addEventListener("pointerdown", activate);
  element.addEventListener("focus", activate);
  element.addEventListener("blur", () => {
    element.classList.remove("is-active");
  });
});

const activateRangeHandle = (handle) => {
  [rangeStartEl, rangeHighlightEl, rangeEndEl].forEach((input) => {
    input.classList.toggle("is-active", input === handle);
  });
};

const updateRangeHandleByIndex = (handle, rawIndex) => {
  const nextIndex = clamp(rawIndex, 0, megaSenaDraws.length - 1);

  if (handle === rangeStartEl) {
    applyRange(Math.min(nextIndex, state.rangeEndIndex), state.rangeEndIndex);
    return;
  }

  if (handle === rangeEndEl) {
    applyRange(state.rangeStartIndex, Math.max(nextIndex, state.rangeStartIndex));
    return;
  }

  setHighlightByIndex(
    clamp(nextIndex, state.rangeStartIndex, state.rangeEndIndex),
    { expandRange: false },
  );
};

if (rangeBarEl) {
  let activeHandle = null;

  const pointerToIndex = (clientX) => {
    const rect = rangeBarEl.getBoundingClientRect();
    if (!rect.width) {
      return state.highlightIndex;
    }
    const ratio = clamp((clientX - rect.left) / rect.width, 0, 1);
    return Math.round(ratio * (megaSenaDraws.length - 1));
  };

  const resolveNearestHandle = (nextIndex) => {
    const handles = [rangeStartEl, rangeHighlightEl, rangeEndEl];
    const positions = new Map([
      [rangeStartEl, state.rangeStartIndex],
      [rangeHighlightEl, state.highlightIndex],
      [rangeEndEl, state.rangeEndIndex],
    ]);

    return handles.reduce((nearest, current) => {
      if (!nearest) {
        return current;
      }
      const currentDistance = Math.abs(positions.get(current) - nextIndex);
      const nearestDistance = Math.abs(positions.get(nearest) - nextIndex);
      return currentDistance < nearestDistance ? current : nearest;
    }, null);
  };

  const handlePointerMove = (event) => {
    if (!activeHandle || !state.showAllLines) {
      return;
    }
    updateRangeHandleByIndex(activeHandle, pointerToIndex(event.clientX));
  };

  const handlePointerUp = () => {
    activeHandle = null;
  };

  rangeBarEl.addEventListener("pointerdown", (event) => {
    if (!state.showAllLines) {
      return;
    }

    const nextIndex = pointerToIndex(event.clientX);
    activeHandle = resolveNearestHandle(nextIndex);
    activateRangeHandle(activeHandle);
    updateRangeHandleByIndex(activeHandle, nextIndex);
    rangeBarEl.setPointerCapture(event.pointerId);
  });

  rangeBarEl.addEventListener("pointermove", handlePointerMove);
  rangeBarEl.addEventListener("pointerup", handlePointerUp);
  rangeBarEl.addEventListener("pointercancel", handlePointerUp);
  rangeBarEl.addEventListener("lostpointercapture", handlePointerUp);
}

dateStartEl.addEventListener("change", (event) => {
  const nextStart = findStartIndexByDate(event.target.value);
  applyRange(Math.min(nextStart, state.rangeEndIndex), state.rangeEndIndex);
});

dateEndEl.addEventListener("change", (event) => {
  const nextEnd = findEndIndexByDate(event.target.value);
  applyRange(state.rangeStartIndex, Math.max(nextEnd, state.rangeStartIndex));
});

backgroundToneRangeEl.addEventListener("input", (event) => {
  state.overlayBackgroundTone = Number(event.target.value);
  updateBackgroundToneControl();
  renderCharts();
});

backgroundToneSwatchEl.addEventListener("input", (event) => {
  state.overlayGradientStart = event.target.value;
  updateBackgroundToneControl();
  renderCharts();
});

toggleAllLinesButton.addEventListener("click", () => {
  state.showAllLines = !state.showAllLines;
  updateAllLinesButton();
  renderCharts();
});

exportPngButton.addEventListener("click", () => {
  const gap = Math.round((window.devicePixelRatio || 1) * 28);
  const width = Math.max(...chartConfigs.map((config) => config.canvasEl.width));
  const height =
    chartConfigs.reduce((sum, config) => sum + config.canvasEl.height, 0) +
    gap * (chartConfigs.length - 1);
  const exportCanvas = document.createElement("canvas");
  exportCanvas.width = width;
  exportCanvas.height = height;
  const exportCtx = exportCanvas.getContext("2d");

  exportCtx.fillStyle = "#ffffff";
  exportCtx.fillRect(0, 0, width, height);

  let currentY = 0;
  chartConfigs.forEach((config, index) => {
    exportCtx.drawImage(config.canvasEl, 0, currentY);
    currentY += config.canvasEl.height;
    if (index < chartConfigs.length - 1) {
      currentY += gap;
    }
  });

  const link = document.createElement("a");
  const concurso = megaSenaDraws[state.highlightIndex]?.concurso ?? "sem-dados";
  link.href = exportCanvas.toDataURL("image/png");
  link.download = `mega-sena-comparativo-concurso-${concurso}.png`;
  link.click();
});

if (!megaSenaDraws.length) {
  setupEmptyPage();
} else {
  setupStats();
  updateDetailPanel();
  updateAllLinesButton();
  resetSorteGame();
  resizeCanvases();

  const resizeObserver = new ResizeObserver(() => resizeCanvases());
  chartConfigs.forEach((config) => resizeObserver.observe(config.shellEl));
}
