/*
 * BGA Terra Mystica arena faction stats collector (browser console)
 *
 * Usage:
 * 1) Log in to BGA
 * 2) Open DevTools on a BGA page (Terra Mystica gamestats page recommended)
 * 3) Paste this whole script into the console and run
 * 4) Read results from window.tmArenaFactionStats
 */

(async () => {
  // -------- Config --------
  const playerId = "93627000"; // kezilu
  const gameId = "1118"; // Terra Mystica
  const startDateStr = "2026-01-13"; // default start date (YYYY-MM-DD, UTC)
  const endDateStr = null; // optional, e.g. "2026-02-21" (YYYY-MM-DD, UTC)
  const concurrency = 6;

  // -------- Helpers --------
  const token = window?.bgaConfig?.requestToken;
  if (!token) throw new Error("No bgaConfig.requestToken found. Run this on a logged-in BGA page.");

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

  const startEpoch = parseDateStartUtc(startDateStr);
  const endEpoch = endDateStr ? parseDateEndUtc(endDateStr) : null;

  if (endEpoch !== null && endEpoch < startEpoch) {
    throw new Error("endDateStr is earlier than startDateStr");
  }

  const post = async (url, data) => {
    const body = new URLSearchParams();
    Object.entries(data).forEach(([k, v]) => body.set(k, String(v)));
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
    const json = await res.json();
    if (!json || json.status !== 1) {
      throw new Error(`${url} failed: ${json?.error || "unknown error"}`);
    }
    return json;
  };

  const rankKey = (r) => (r === 1 ? "first" : r === 2 ? "second" : r === 3 ? "third" : r === 4 ? "fourth" : "other");

  const emptyStat = () => ({
    games: 0,
    first: 0,
    second: 0,
    third: 0,
    fourth: 0,
    other: 0,
    rank_sum: 0,
  });

  const toSummaryArray = (obj) =>
    Object.entries(obj)
      .map(([faction, s]) => ({
        faction,
        games: s.games,
        first: s.first,
        second: s.second,
        third: s.third,
        fourth: s.fourth,
        other: s.other,
        avg_rank: s.games ? Number((s.rank_sum / s.games).toFixed(3)) : null,
      }))
      .sort((a, b) => b.games - a.games || a.faction.localeCompare(b.faction));

  const toUtc = (epoch) => new Date(epoch * 1000).toISOString().replace("T", " ").replace(".000Z", " UTC");

  // -------- 1) Load all relevant gamestats rows --------
  console.log("[TM] Loading games...");
  const allRows = [];

  const first = await post("/gamestats/gamestats/getGames.html", {
    player: playerId,
    opponent_id: 0,
    game_id: gameId,
    finished: 0,
    updateStats: 1,
  });
  allRows.push(...(first.data?.tables || []));

  for (let page = 2; page <= 300; page++) {
    const resp = await post("/gamestats/gamestats/getGames.html", {
      player: playerId,
      opponent_id: 0,
      game_id: gameId,
      finished: 0,
      updateStats: 0,
      page,
    });

    const rows = resp.data?.tables || [];
    if (!rows.length) break;
    allRows.push(...rows);

    // Rows are newest -> oldest. Stop once this page is older than start date.
    const oldest = Math.min(...rows.map((r) => Number(r.end)));
    if (oldest < startEpoch) break;
  }

  const inRangeArena = allRows.filter((r) => {
    const end = Number(r.end);
    return r.arena_win !== null && end >= startEpoch && (endEpoch === null || end <= endEpoch);
  });

  console.log("[TM] Arena games in range:", inRangeArena.length);

  // -------- 2) Load tableinfos for faction data --------
  const tableIds = [...new Set(inRangeArena.map((r) => String(r.table_id)))];
  const tableInfoMap = new Map();

  console.log("[TM] Loading tableinfos...");
  let ptr = 0;
  async function worker() {
    while (ptr < tableIds.length) {
      const i = ptr++;
      const id = tableIds[i];
      try {
        const resp = await post("/table/table/tableinfos.html", { id });
        tableInfoMap.set(id, resp.data);
      } catch (e) {
        console.warn(`[TM] tableinfos failed for ${id}:`, e.message);
      }
    }
  }
  await Promise.all(Array.from({ length: Math.min(concurrency, tableIds.length) }, () => worker()));

  // -------- 3) Compile stats --------
  const factionOverall = {};
  const factionForPlayer = {};
  const perGamePlayer = [];

  let totalFactionValues = 0;
  let maskedFactionValues = 0;

  for (const row of inRangeArena) {
    const tableId = String(row.table_id);
    const info = tableInfoMap.get(tableId);

    const pf = info?.result?.stats?.player?.player_faction;
    const values = pf?.values || {};
    const labels = pf?.valuelabels || {};

    const players = String(row.players || "").split(",");
    const ranks = String(row.ranks || "").split(",").map((x) => Number(x));
    const scores = String(row.scores || "").split(",").map((x) => Number(x));
    const endEpochRow = Number(row.end);

    for (let i = 0; i < players.length; i++) {
      const pid = players[i];
      const rank = ranks[i];
      let faction = labels[pid] ?? values[pid] ?? "(unknown)";

      totalFactionValues++;
      if (faction === "*masked*") {
        maskedFactionValues++;
        faction = "(masked)";
      }

      if (!factionOverall[faction]) factionOverall[faction] = emptyStat();
      factionOverall[faction].games++;
      factionOverall[faction][rankKey(rank)]++;
      factionOverall[faction].rank_sum += rank;

      if (pid === playerId) {
        if (!factionForPlayer[faction]) factionForPlayer[faction] = emptyStat();
        factionForPlayer[faction].games++;
        factionForPlayer[faction][rankKey(rank)]++;
        factionForPlayer[faction].rank_sum += rank;

        perGamePlayer.push({
          endEpoch: endEpochRow,
          endUtc: toUtc(endEpochRow),
          tableId,
          url: `https://boardgamearena.com/table?table=${tableId}`,
          faction,
          rank,
          score: scores[i],
        });
      }
    }
  }

  perGamePlayer.sort((a, b) => b.endEpoch - a.endEpoch);

  const rankCounts = perGamePlayer.reduce((acc, g) => {
    acc[g.rank] = (acc[g.rank] || 0) + 1;
    return acc;
  }, {});
  const games = perGamePlayer.length;
  const wins = rankCounts[1] || 0;
  const top2 = (rankCounts[1] || 0) + (rankCounts[2] || 0);
  const top3 = top2 + (rankCounts[3] || 0);

  const result = {
    generatedAtUtc: new Date().toISOString(),
    startDateUtc: `${startDateStr} 00:00:00 UTC`,
    endDateUtc: endDateStr ? `${endDateStr} 23:59:59 UTC` : null,
    totalArenaGames: inRangeArena.length,
    playerArenaGames: games,
    playerPlacementCounts: { "1st": rankCounts[1] || 0, "2nd": rankCounts[2] || 0, "3rd": rankCounts[3] || 0, "4th": rankCounts[4] || 0 },
    playerWinRate: games ? wins / games : 0,
    playerTop2Rate: games ? top2 / games : 0,
    playerTop3Rate: games ? top3 / games : 0,
    factionOverall: toSummaryArray(factionOverall),
    factionForPlayer: toSummaryArray(factionForPlayer),
    perGamePlayer,
    masking: {
      totalFactionValues,
      maskedFactionValues,
      allMasked: totalFactionValues > 0 && maskedFactionValues === totalFactionValues,
    },
  };

  window.tmArenaFactionStats = result;

  console.log("[TM] Done. Result is in window.tmArenaFactionStats");
  console.log("[TM] Player placement counts:", result.playerPlacementCounts);
  console.table(result.factionForPlayer);
  console.table(result.factionOverall);
  console.table(result.perGamePlayer.slice(0, 30));

  if (result.masking.allMasked) {
    console.warn("[TM] Factions are masked in your current session (likely premium/visibility limitation).");
  }

  // Optional quick CSV download helper.
  window.downloadTmArenaPlayerGamesCsv = () => {
    const rows = [
      ["end_utc", "table_id", "url", "faction", "rank", "score"],
      ...result.perGamePlayer.map((g) => [g.endUtc, g.tableId, g.url, g.faction, g.rank, g.score]),
    ];
    const csv = rows.map((r) => r.map((x) => `"${String(x).replace(/"/g, '""')}"`).join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = "tm_arena_player_games_by_faction.csv";
    document.body.appendChild(a);
    a.click();
    a.remove();
  };

  return result;
})();
