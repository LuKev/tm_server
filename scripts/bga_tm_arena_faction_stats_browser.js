/*
 * BGA Terra Mystica arena batch faction stats collector (browser console)
 *
 * Usage:
 * 1) Log in to BGA in this browser.
 * 2) Open DevTools on any BGA page.
 * 3) Paste this whole script into the console and run.
 * 4) Wait for "[TM] Done".
 * 5) Use:
 *    - window.tmBatchFactionStats
 *    - downloadTmBatchFactionSummaryCsv()
 *    - downloadTmBatchFactionRecordsCsv()
 */

(async () => {
  const config = {
    gameId: "1118", // Terra Mystica
    startDateStr: "2026-01-13", // YYYY-MM-DD (UTC)
    endDateStr: null, // optional YYYY-MM-DD (UTC)
    arenaOnly: true,
    maxPagesPerPlayer: 350,
    requestDelayMs: 450, // global delay between HTTP requests
    parallelism: 1, // tableinfos workers
    includeAllPlayersFromMatchedGames: true, // true => aggregate all players in matched games, not only tracked IDs
    excludeAllZeroOrOneFinalScoreGames: true, // filter likely-abandoned games where everyone ends on 0/1 VP
    hardcodedPlayerIds: [
      "92355390", "96550044", "93627000", "91336208", "84208794", "90408684", "89891647", "28792375", "98682054", "90386040",
      "84479821", "88676986", "95728036", "88239113", "85141334", "85003379", "94150245", "92667050", "88328712", "85646799",
      "87425338", "90303102", "28163152", "96969637", "95537516", "86950199", "93130593", "96401697", "84191173", "88076867",
      "84712163", "85367922", "94542047", "84188299", "85166337", "85444399", "97114695", "88099285", "86984785", "29333422",
      "86125352", "84844638", "84849940", "93979478", "89668228", "92737541", "94987548", "91695432", "84384226", "89625033",
      "84938448", "84398565", "87157341", "92375485", "89801317", "95947629", "87123090", "97747092", "85395108", "98216248",
      "87462235", "84960351", "92244566", "4463512", "85746561", "96747522", "97979059", "92086225", "93228302", "85191627",
    ],
  };

  const token = window?.bgaConfig?.requestToken;
  if (!token) {
    throw new Error("No bgaConfig.requestToken found. Run this on a logged-in BGA page.");
  }

  const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

  const parseDateStartUtc = (s) => {
    const d = new Date(`${s}T00:00:00Z`);
    if (Number.isNaN(d.getTime())) throw new Error(`Invalid startDateStr: ${s}`);
    return Math.floor(d.getTime() / 1000);
  };

  const parseDateEndUtc = (s) => {
    const d = new Date(`${s}T23:59:59Z`);
    if (Number.isNaN(d.getTime())) throw new Error(`Invalid endDateStr: ${s}`);
    return Math.floor(d.getTime() / 1000);
  };

  const startEpoch = parseDateStartUtc(config.startDateStr);
  const endEpoch = config.endDateStr ? parseDateEndUtc(config.endDateStr) : null;
  if (endEpoch !== null && endEpoch < startEpoch) {
    throw new Error("endDateStr is earlier than startDateStr");
  }

  const trackedPlayerIds = [...new Set(config.hardcodedPlayerIds.map((x) => String(x).trim()).filter(Boolean))];
  const trackedPlayerSet = new Set(trackedPlayerIds);

  let nextRequestAt = 0;

  const waitForRateLimit = async () => {
    const now = Date.now();
    if (nextRequestAt > now) {
      await sleep(nextRequestAt - now);
    }
    nextRequestAt = Date.now() + config.requestDelayMs;
  };

  const post = async (url, data, retries = 2) => {
    const body = new URLSearchParams();
    Object.entries(data).forEach(([k, v]) => body.set(k, String(v)));

    let lastError = null;
    for (let attempt = 0; attempt <= retries; attempt++) {
      try {
        await waitForRateLimit();
        const res = await fetch(url, {
          method: "POST",
          credentials: "include",
          headers: {
            "Content-Type": "application/x-www-form-urlencoded; charset=UTF-8",
            "X-Requested-With": "XMLHttpRequest",
            "X-Request-Token": token,
          },
          body,
        });

        const text = await res.text();
        let json;
        try {
          json = JSON.parse(text);
        } catch {
          throw new Error(`Non-JSON response (${res.status})`);
        }

        if (!json || json.status !== 1) {
          throw new Error(json?.error || `status=${json?.status}`);
        }

        return json;
      } catch (err) {
        lastError = err;
        if (attempt < retries) {
          await sleep(500 * (attempt + 1));
        }
      }
    }

    throw new Error(`${url} failed after retries: ${lastError?.message || "unknown error"}`);
  };

  const splitCsv = (s) => (s == null || s === "" ? [] : String(s).split(",").map((x) => x.trim()));

  const toNumber = (v) => {
    if (v === null || v === undefined || v === "") return null;
    if (typeof v === "number") return Number.isFinite(v) ? v : null;

    const normalized = String(v)
      .replace(/\u00a0/g, " ")
      .replace(/\s+/g, "")
      .replace(/,/g, ".");

    const direct = Number(normalized);
    if (Number.isFinite(direct)) return direct;

    const m = normalized.match(/-?\d+(?:\.\d+)?/);
    return m ? Number(m[0]) : null;
  };

  const toUtc = (epoch) => new Date(epoch * 1000).toISOString().replace("T", " ").replace(".000Z", " UTC");

  const rankKey = (r) => (r === 1 ? "first" : r === 2 ? "second" : r === 3 ? "third" : r === 4 ? "fourth" : "other");

  const emptyAgg = () => ({
    games: 0,
    first: 0,
    second: 0,
    third: 0,
    fourth: 0,
    other: 0,
    rankSum: 0,
    rankCount: 0,
    startingVpSum: 0,
    startingVpCount: 0,
    totalVpAwardedSum: 0,
    totalVpAwardedCount: 0,
    totalVpGainedSum: 0,
    totalVpGainedCount: 0,
    totalPointsSum: 0,
    totalPointsCount: 0,
  });

  const avg = (sum, count) => (count ? sum / count : null);

  const round3 = (n) => (n == null ? null : Number(n.toFixed(3)));

  const getPlayerId = (playerNode, fallbackKey = null) => {
    const id = playerNode?.id ?? playerNode?.player_id ?? playerNode?.playerId ?? playerNode?.userid ?? playerNode?.user_id ?? playerNode?.uid ?? fallbackKey;
    return id == null ? null : String(id);
  };

  const normalizeStatEntries = (tableInfoData) => {
    const playerStats = tableInfoData?.result?.stats?.player ?? tableInfoData?.result?.stats?.players ?? {};
    const entries = [];
    if (Array.isArray(playerStats)) {
      playerStats.forEach((stat, idx) => entries.push({ key: `idx_${idx}`, stat }));
      return entries;
    }
    if (playerStats && typeof playerStats === "object") {
      Object.entries(playerStats).forEach(([key, stat]) => entries.push({ key, stat }));
    }
    return entries;
  };

  const findStat = (statEntries, predicate) => {
    for (const entry of statEntries) {
      const key = String(entry.key || "").toLowerCase();
      const name = String(entry.stat?.name ?? entry.stat?.label ?? entry.stat?.title ?? "").toLowerCase();
      const desc = String(entry.stat?.description ?? entry.stat?.help ?? "").toLowerCase();
      if (predicate({ key, name, desc })) {
        return entry.stat;
      }
    }
    return null;
  };

  const getValueByPidOrIndex = (obj, pid, indexHint) => {
    if (!obj) return null;
    if (Array.isArray(obj)) {
      if (indexHint == null || indexHint < 0 || indexHint >= obj.length) return null;
      return obj[indexHint];
    }
    if (typeof obj !== "object") return null;
    const pidStr = String(pid);
    const pidNumStr = String(Number(pid));
    if (obj[pidStr] !== undefined) return obj[pidStr];
    if (obj[pidNumStr] !== undefined) return obj[pidNumStr];
    return null;
  };

  const buildTablePlayers = (tableInfoData) => {
    const players = [];
    const playerNode = tableInfoData?.result?.player ?? tableInfoData?.result?.players;

    if (Array.isArray(playerNode)) {
      playerNode.forEach((player, idx) => {
        const pid = getPlayerId(player, null);
        if (pid != null) {
          players.push({ pid, idx, score: toNumber(player?.score), rank: toNumber(player?.rank) });
        }
      });
      return players;
    }

    if (playerNode && typeof playerNode === "object") {
      Object.entries(playerNode).forEach(([key, player], idx) => {
        if (!player || typeof player !== "object") return;
        const pid = getPlayerId(player, key);
        if (pid != null) {
          players.push({ pid, idx, score: toNumber(player?.score), rank: toNumber(player?.rank) });
        }
      });
    }

    return players;
  };

  const extractTableDerived = (tableInfoData, rowPlayers) => {
    const statEntries = normalizeStatEntries(tableInfoData);
    const tablePlayers = buildTablePlayers(tableInfoData);

    const rowIndexByPid = new Map();
    rowPlayers.forEach((pid, idx) => rowIndexByPid.set(String(pid), idx));

    const tableIndexByPid = new Map();
    const scoreMap = new Map();
    const rankMap = new Map();
    tablePlayers.forEach((p) => {
      tableIndexByPid.set(String(p.pid), p.idx);
      if (Number.isFinite(p.score)) scoreMap.set(String(p.pid), p.score);
      if (Number.isFinite(p.rank)) rankMap.set(String(p.pid), p.rank);
    });

    const factionStat =
      findStat(statEntries, (x) => x.key.includes("player_faction") || x.key === "faction") ||
      findStat(statEntries, (x) => x.name.includes("faction chosen") || (x.name.includes("faction") && x.name.includes("chosen")));

    const startingVpStat =
      findStat(statEntries, (x) => x.key.includes("vpby_faction")) ||
      findStat(statEntries, (x) => x.name.includes("starting vp") && x.name.includes("faction"));

    const totalVpAwardedStat =
      findStat(statEntries, (x) => x.key.includes("vpby_total")) ||
      findStat(statEntries, (x) => x.name.includes("total vp awarded"));

    const vpSpentAbilityStat =
      findStat(statEntries, (x) => x.key.includes("vp_spent_ability")) ||
      findStat(statEntries, (x) => x.name.includes("vp spent with faction ability"));

    const vpSpentStructStat =
      findStat(statEntries, (x) => x.key.includes("vp_spent") && !x.key.includes("ability")) ||
      findStat(statEntries, (x) => x.name.includes("vp lost to gain power via structures"));

    const getStatValue = (stat, pid, preferLabel = false) => {
      if (!stat) return null;
      const pidStr = String(pid);
      const indexHint = tableIndexByPid.get(pidStr) ?? rowIndexByPid.get(pidStr) ?? null;

      if (preferLabel) {
        const label = getValueByPidOrIndex(stat.valuelabels, pidStr, indexHint);
        if (label !== null) return label;
        return getValueByPidOrIndex(stat.values, pidStr, indexHint);
      }

      const value = getValueByPidOrIndex(stat.values, pidStr, indexHint);
      if (value !== null) return value;
      return getValueByPidOrIndex(stat.valuelabels, pidStr, indexHint);
    };

    return {
      scoreMap,
      rankMap,
      tablePlayerIds: tablePlayers.map((p) => String(p.pid)),
      getFaction: (pid) => {
        let faction = getStatValue(factionStat, pid, true);
        if (faction == null || faction === "") return "(unknown)";
        faction = String(faction);
        if (faction === "*masked*") return "(masked)";
        return faction;
      },
      getStartingVp: (pid) => toNumber(getStatValue(startingVpStat, pid, false)),
      getTotalVpAwarded: (pid) => toNumber(getStatValue(totalVpAwardedStat, pid, false)),
      getVpSpentStructures: (pid) => toNumber(getStatValue(vpSpentStructStat, pid, false)),
      getVpSpentAbility: (pid) => toNumber(getStatValue(vpSpentAbilityStat, pid, false)),
    };
  };

  const computeMidRanksByScore = (scoreByPid, orderedPlayerIds) => {
    const out = new Map();
    const players = orderedPlayerIds.map((pid) => ({ pid: String(pid), score: scoreByPid.get(String(pid)) }));
    if (!players.length) return out;
    if (players.some((p) => !Number.isFinite(p.score))) return out;

    players.sort((a, b) => b.score - a.score || a.pid.localeCompare(b.pid));

    let pos = 1;
    for (let i = 0; i < players.length; ) {
      let j = i + 1;
      while (j < players.length && players[j].score === players[i].score) j++;
      const count = j - i;
      const midRank = (pos + (pos + count - 1)) / 2;
      for (let k = i; k < j; k++) {
        out.set(players[k].pid, midRank);
      }
      pos += count;
      i = j;
    }
    return out;
  };

  const toCsv = (rows) => rows.map((row) => row.map((v) => `"${String(v ?? "").replace(/"/g, '""')}"`).join(",")).join("\n");
  let lastCsvBlobUrl = null;

  const triggerCsvDownload = async (filename, rows) => {
    if (!rows || !rows.length) {
      console.warn("[TM] No CSV rows to download.");
      return;
    }

    const csv = toCsv(rows);
    if (typeof window.showSaveFilePicker === "function") {
      try {
        const handle = await window.showSaveFilePicker({
          suggestedName: filename,
          types: [{ description: "CSV file", accept: { "text/csv": [".csv"] } }],
        });
        const writable = await handle.createWritable();
        await writable.write(csv);
        await writable.close();
        console.log(`[TM] Saved ${filename} via file picker.`);
        return;
      } catch (err) {
        if (err?.name !== "AbortError") {
          console.warn("[TM] File picker save failed, falling back to blob URL:", err?.message || err);
        }
      }
    }

    const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);

    if (lastCsvBlobUrl) {
      URL.revokeObjectURL(lastCsvBlobUrl);
    }
    lastCsvBlobUrl = url;
    window.tmLastCsvBlobUrl = url;
    window.tmOpenLastCsvBlobUrl = () => window.open(window.tmLastCsvBlobUrl, "_blank", "noopener,noreferrer");

    // Prefer opening the blob URL directly (avoids BGA click interception on in-page links).
    const opened = window.open(url, "_blank", "noopener,noreferrer");

    // Secondary attempt: native download attribute.
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.rel = "noopener noreferrer";
    a.target = "_blank";
    a.style.display = "none";
    a.addEventListener("click", (evt) => evt.stopPropagation(), true);
    document.documentElement.appendChild(a);
    a.click();
    a.remove();

    console.log(
      opened
        ? `[TM] Opened ${filename} in a new tab (or triggered download).`
        : "[TM] Download may be blocked. Run tmOpenLastCsvBlobUrl() and save from that tab."
    );
  };

  const loadGamesForPlayer = async (playerId) => {
    const rows = [];

    const first = await post("/gamestats/gamestats/getGames.html", {
      player: playerId,
      opponent_id: 0,
      game_id: config.gameId,
      finished: 0,
      updateStats: 1,
    });

    rows.push(...(first.data?.tables || []));

    for (let page = 2; page <= config.maxPagesPerPlayer; page++) {
      const resp = await post("/gamestats/gamestats/getGames.html", {
        player: playerId,
        opponent_id: 0,
        game_id: config.gameId,
        finished: 0,
        updateStats: 0,
        page,
      });

      const pageRows = resp.data?.tables || [];
      if (!pageRows.length) break;

      rows.push(...pageRows);

      const oldest = Math.min(...pageRows.map((r) => Number(r.end || 0)).filter((n) => Number.isFinite(n) && n > 0));
      if (oldest && oldest < startEpoch) break;
    }

    return rows;
  };

  console.log("[TM] Starting batch collection...");
  console.log("[TM] Players:", trackedPlayerIds.length, "| Start:", config.startDateStr, "| End:", config.endDateStr || "(none)");

  const gameRowsByTableId = new Map();
  const failedPlayers = [];
  let scannedRows = 0;

  for (let i = 0; i < trackedPlayerIds.length; i++) {
    const pid = trackedPlayerIds[i];
    try {
      const rows = await loadGamesForPlayer(pid);
      scannedRows += rows.length;

      for (const row of rows) {
        const tid = row?.table_id != null ? String(row.table_id) : null;
        const ended = Number(row?.end || 0);
        if (!tid || !ended) continue;

        const isArena = row.arena_win !== null && row.arena_win !== undefined;
        if (config.arenaOnly && !isArena) continue;
        if (ended < startEpoch) continue;
        if (endEpoch !== null && ended > endEpoch) continue;

        if (!gameRowsByTableId.has(tid)) {
          gameRowsByTableId.set(tid, row);
        }
      }

      if ((i + 1) % 10 === 0 || i + 1 === trackedPlayerIds.length) {
        console.log(`[TM] Loaded players ${i + 1}/${trackedPlayerIds.length}. Unique games so far: ${gameRowsByTableId.size}`);
      }
    } catch (err) {
      failedPlayers.push({ playerId: pid, error: err.message });
      console.warn(`[TM] Failed loading games for player ${pid}:`, err.message);
    }
  }

  const tableIds = [...gameRowsByTableId.keys()];
  console.log(`[TM] Unique arena games in date range: ${tableIds.length}`);

  const tableInfoMap = new Map();
  const failedTableInfos = [];
  let tableCursor = 0;

  const worker = async () => {
    while (tableCursor < tableIds.length) {
      const idx = tableCursor++;
      const tableId = tableIds[idx];
      try {
        const resp = await post("/table/table/tableinfos.html", { id: tableId }, 2);
        tableInfoMap.set(tableId, resp.data);
      } catch (err) {
        failedTableInfos.push({ tableId, error: err.message });
      }

      if ((idx + 1) % 50 === 0 || idx + 1 === tableIds.length) {
        console.log(`[TM] Loaded tableinfos ${idx + 1}/${tableIds.length}`);
      }
    }
  };

  const workers = Array.from({ length: Math.min(config.parallelism, Math.max(1, tableIds.length)) }, () => worker());
  await Promise.all(workers);

  const records = [];
  const filteredAllZeroOrOneGames = [];

  for (const tableId of tableIds) {
    const row = gameRowsByTableId.get(tableId);
    const ended = Number(row?.end || 0);

    const rowPlayers = splitCsv(row?.players);
    const rowRanks = splitCsv(row?.ranks).map((x) => toNumber(x));
    const rowScores = splitCsv(row?.scores).map((x) => toNumber(x));

    const rankByPid = new Map();
    const scoreByPid = new Map();
    for (let i = 0; i < rowPlayers.length; i++) {
      const pid = String(rowPlayers[i]);
      rankByPid.set(pid, rowRanks[i] == null ? null : Number(rowRanks[i]));
      if (Number.isFinite(rowScores[i])) {
        scoreByPid.set(pid, rowScores[i]);
      }
    }

    const infoData = tableInfoMap.get(tableId) || null;
    const derived = infoData ? extractTableDerived(infoData, rowPlayers) : null;
    if (derived?.scoreMap) {
      for (const [pid, score] of derived.scoreMap.entries()) {
        if (Number.isFinite(score)) {
          scoreByPid.set(String(pid), score);
        }
      }
    }

    const allGamePlayerIds = [...new Set([...rowPlayers.map((pid) => String(pid)), ...(derived?.tablePlayerIds || [])])];
    const hasAllScoresForGame = allGamePlayerIds.length > 0 && allGamePlayerIds.every((pid) => Number.isFinite(scoreByPid.get(String(pid))));
    const isAllZeroOrOneScoreGame =
      hasAllScoresForGame &&
      allGamePlayerIds.every((pid) => {
        const score = scoreByPid.get(String(pid));
        return score === 0 || score === 1;
      });

    if (config.excludeAllZeroOrOneFinalScoreGames && isAllZeroOrOneScoreGame) {
      filteredAllZeroOrOneGames.push(tableId);
      continue;
    }

    const playerIdsToProcess = config.includeAllPlayersFromMatchedGames
      ? allGamePlayerIds
      : allGamePlayerIds.filter((pid) => trackedPlayerSet.has(String(pid)));
    const adjustedRankByPid = computeMidRanksByScore(scoreByPid, allGamePlayerIds);

    for (const pidRaw of playerIdsToProcess) {
      const pid = String(pidRaw);

      const rankRaw = rankByPid.get(pid) ?? derived?.rankMap?.get(pid) ?? null;
      const rankForAverage = adjustedRankByPid.get(pid) ?? (Number.isFinite(rankRaw) ? Number(rankRaw) : null);
      const totalPointsScored = scoreByPid.get(pid) ?? null;

      const faction = derived?.getFaction(pid) ?? "(unknown)";
      const startingVp = derived?.getStartingVp(pid) ?? null;
      const totalVpAwarded = derived?.getTotalVpAwarded(pid) ?? null;
      const vpSpentStructures = derived?.getVpSpentStructures(pid) ?? null;
      const vpSpentAbility = derived?.getVpSpentAbility(pid) ?? null;
      const totalVpGained =
        totalVpAwarded == null
          ? null
          : totalVpAwarded - (vpSpentStructures == null ? 0 : vpSpentStructures) - (vpSpentAbility == null ? 0 : vpSpentAbility);

      records.push({
        tableId,
        tableUrl: `https://boardgamearena.com/table?table=${tableId}`,
        endEpoch: ended,
        endUtc: toUtc(ended),
        playerId: pid,
        rank: rankRaw,
        rankRaw,
        rankForAverage,
        isTrackedPlayer: trackedPlayerSet.has(pid),
        faction,
        startingVp,
        totalVpAwarded,
        vpSpentStructures,
        vpSpentAbility,
        totalVpGained,
        totalPointsScored,
      });
    }
  }

  records.sort((a, b) => b.endEpoch - a.endEpoch || a.playerId.localeCompare(b.playerId));

  const aggByFaction = {};
  for (const r of records) {
    const faction = r.faction || "(unknown)";
    if (!aggByFaction[faction]) aggByFaction[faction] = emptyAgg();

    const agg = aggByFaction[faction];
    agg.games += 1;

    if (Number.isFinite(r.rankRaw)) {
      const rawRank = Number(r.rankRaw);
      if (rawRank === 1 || rawRank === 2 || rawRank === 3 || rawRank === 4) {
        agg[rankKey(rawRank)] += 1;
      } else {
        agg.other += 1;
      }
    } else {
      agg.other += 1;
    }

    if (Number.isFinite(r.rankForAverage)) {
      const avgRankValue = Number(r.rankForAverage);
      agg.rankSum += avgRankValue;
      agg.rankCount += 1;
    }

    if (Number.isFinite(r.startingVp)) {
      agg.startingVpSum += r.startingVp;
      agg.startingVpCount += 1;
    }

    if (Number.isFinite(r.totalVpAwarded)) {
      agg.totalVpAwardedSum += r.totalVpAwarded;
      agg.totalVpAwardedCount += 1;
    }

    if (Number.isFinite(r.totalVpGained)) {
      agg.totalVpGainedSum += r.totalVpGained;
      agg.totalVpGainedCount += 1;
    }

    if (Number.isFinite(r.totalPointsScored)) {
      agg.totalPointsSum += r.totalPointsScored;
      agg.totalPointsCount += 1;
    }
  }

  const summaryByFaction = Object.entries(aggByFaction)
    .map(([faction, s]) => ({
      faction,
      frequency: s.games,
      first: s.first,
      second: s.second,
      third: s.third,
      fourth: s.fourth,
      other: s.other,
      avgPlacement: round3(avg(s.rankSum, s.rankCount)),
      avgStartingVpChosenFaction: round3(avg(s.startingVpSum, s.startingVpCount)),
      avgTotalVpAwarded: round3(avg(s.totalVpAwardedSum, s.totalVpAwardedCount)),
      avgTotalVpGained: round3(avg(s.totalVpGainedSum, s.totalVpGainedCount)),
      avgTotalPointsScored: round3(avg(s.totalPointsSum, s.totalPointsCount)),
    }))
    .sort((a, b) => b.frequency - a.frequency || (a.avgPlacement ?? 999) - (b.avgPlacement ?? 999) || a.faction.localeCompare(b.faction));

  const result = {
    generatedAtUtc: new Date().toISOString(),
    config: {
      gameId: config.gameId,
      startDateUtc: `${config.startDateStr} 00:00:00 UTC`,
      endDateUtc: config.endDateStr ? `${config.endDateStr} 23:59:59 UTC` : null,
      arenaOnly: config.arenaOnly,
      requestDelayMs: config.requestDelayMs,
      parallelism: config.parallelism,
      includeAllPlayersFromMatchedGames: config.includeAllPlayersFromMatchedGames,
      excludeAllZeroOrOneFinalScoreGames: config.excludeAllZeroOrOneFinalScoreGames,
      trackedPlayerCount: trackedPlayerIds.length,
      avgPlacementMethod: "mid-rank by score ties (e.g., tie for 1st => 1.5)",
    },
    scannedRows,
    discoveredUniqueGames: tableIds.length,
    filteredOutAllZeroOrOneGames: filteredAllZeroOrOneGames.length,
    uniqueGames: tableIds.length - filteredAllZeroOrOneGames.length,
    recordsCount: records.length,
    failedPlayers,
    failedTableInfos,
    filteredAllZeroOrOneGameIds: filteredAllZeroOrOneGames,
    trackedPlayerIds,
    summaryByFaction,
    records,
  };

  window.tmBatchFactionStats = result;

  window.downloadTmBatchFactionSummaryCsv = () => {
    const data = window.tmBatchFactionStats;
    if (!data?.summaryByFaction) {
      console.warn("[TM] tmBatchFactionStats not found. Run the collector script first.");
      return;
    }

    const rows = [
      [
        "faction",
        "frequency",
        "first",
        "second",
        "third",
        "fourth",
        "other",
        "avg_placement",
        "avg_starting_vp_chosen_faction",
        "avg_total_vp_awarded",
        "avg_total_vp_gained",
        "avg_total_points_scored",
      ],
      ...data.summaryByFaction.map((s) => [
        s.faction,
        s.frequency,
        s.first,
        s.second,
        s.third,
        s.fourth,
        s.other,
        s.avgPlacement,
        s.avgStartingVpChosenFaction,
        s.avgTotalVpAwarded,
        s.avgTotalVpGained,
        s.avgTotalPointsScored,
      ]),
    ];

    triggerCsvDownload("tm_batch_faction_summary.csv", rows);
  };

  window.downloadTmBatchFactionRecordsCsv = () => {
    const data = window.tmBatchFactionStats;
    if (!data?.records) {
      console.warn("[TM] tmBatchFactionStats not found. Run the collector script first.");
      return;
    }

    const rows = [
      [
        "end_utc",
        "table_id",
        "table_url",
        "player_id",
        "is_tracked_player",
        "rank",
        "rank_raw",
        "rank_for_average",
        "faction",
        "starting_vp",
        "total_vp_awarded",
        "vp_spent_structures",
        "vp_spent_ability",
        "total_vp_gained",
        "total_points_scored",
      ],
      ...data.records.map((r) => [
        r.endUtc,
        r.tableId,
        r.tableUrl,
        r.playerId,
        r.isTrackedPlayer,
        r.rank,
        r.rankRaw,
        r.rankForAverage,
        r.faction,
        r.startingVp,
        r.totalVpAwarded,
        r.vpSpentStructures,
        r.vpSpentAbility,
        r.totalVpGained,
        r.totalPointsScored,
      ]),
    ];

    triggerCsvDownload("tm_batch_faction_records.csv", rows);
  };

  console.log("[TM] Done. Result -> window.tmBatchFactionStats");
  console.log("[TM] CSV helpers ready: downloadTmBatchFactionSummaryCsv(), downloadTmBatchFactionRecordsCsv()");
  console.log("[TM] Summary sample:");
  console.table(summaryByFaction.slice(0, 20));

  if (failedPlayers.length) {
    console.warn(`[TM] Failed player lookups: ${failedPlayers.length}`);
  }
  if (failedTableInfos.length) {
    console.warn(`[TM] Failed tableinfos: ${failedTableInfos.length}`);
  }
  if (filteredAllZeroOrOneGames.length) {
    console.warn(`[TM] Filtered all-0/1-VP games: ${filteredAllZeroOrOneGames.length}`);
  }

  return result;
})();
