package gamestats

/** Análisis funcional: filter/map/flatMap/groupBy/foldLeft/reduceOption/collect + composición. */
object Analytics:

  private def ordered(events: List[Event]): List[Event] =
    events.sortBy(e => (e.timestamp, e.eventId))

  private def dataNum(e: Event, k: String): Option[Double] =
    e.data.fields.get(k) match
      case Some(JNum(n)) => Some(n)
      case Some(JStr(s)) => s.toDoubleOption
      case _             => None

  private def dataStr(e: Event, k: String): String =
    e.data.fields.get(k) match
      case Some(JStr(s)) => s
      case Some(JNum(n)) => if n.isWhole then n.toLong.toString else n.toString
      case _             => ""

  // ---------- Racing ----------
  def racingStats(events: List[Event]): JObj =
    val laps = events.filter(_.eventType == "LAP_COMPLETED")
    val lapTimes: List[Double] = laps.flatMap(e => dataNum(e, "lapTimeMs"))
    val best: Option[Double] = lapTimes.reduceOption(_ min _)
    val (sum, n) = lapTimes.foldLeft((0.0, 0)) { case ((s, c), t) => (s + t, c + 1) }
    val avg = if n > 0 then sum / n else 0.0
    val penalties = events.count(_.eventType == "PENALTY_RECEIVED")
    // Evolución global de posiciones: POSITION_CHANGED.toPosition ++ LAP_COMPLETED.position, en orden
    val evolution: List[Double] = ordered(events).flatMap { e =>
      e.eventType match
        case "POSITION_CHANGED" => dataNum(e, "toPosition").toList
        case "LAP_COMPLETED"    => dataNum(e, "position").toList
        case _                  => Nil
    }
    val players = events.map(_.playerId).filter(_.nonEmpty).distinct.size
    JObj(Map(
      "lapsCompleted"     -> JNum(laps.size.toDouble),
      "bestLapTime"       -> best.map(JNum(_)).getOrElse(JNull),
      "averageLapTime"    -> JNum(avg),
      "penalties"         -> JNum(penalties.toDouble),
      "positionEvolution" -> JArr(evolution.map(JNum(_))),
      "playersCount"      -> JNum(players.toDouble)
    ))

  def racingByPlayer(events: List[Event]): JObj =
    val byPlayer: Map[String, List[Event]] =
      events.filter(_.playerId.nonEmpty).groupBy(_.playerId)
    JObj(byPlayer.map { (pid, evs) =>
      val laps = evs.filter(_.eventType == "LAP_COMPLETED")
      val times = laps.flatMap(e => dataNum(e, "lapTimeMs"))
      val best = times.reduceOption(_ min _)
      val (sum, n) = times.foldLeft((0.0, 0)) { case ((s, c), t) => (s + t, c + 1) }
      val pens = evs.count(_.eventType == "PENALTY_RECEIVED")
      val positions = ordered(evs).collect {
        case e if e.eventType == "POSITION_CHANGED" => dataNum(e, "toPosition")
        case e if e.eventType == "LAP_COMPLETED"    => dataNum(e, "position")
      }.flatten
      pid -> JObj(Map(
        "totalLaps"    -> JNum(laps.size.toDouble),
        "bestLap"      -> best.map(JNum(_)).getOrElse(JNull),
        "averageLap"   -> JNum(if n > 0 then sum / n else 0.0),
        "penalties"    -> JNum(pens.toDouble),
        "positions"    -> JArr(positions.map(JNum(_)))
      ))
    })

  /** Remontada: primera posición >= 3 y última mejorada (<= 2 o mejora >= 2), con >= 2 evidencias. */
  def detectComebacks(events: List[Event]): List[JObj] =
    val byPlayer = events.filter(_.playerId.nonEmpty).groupBy(_.playerId)
    byPlayer.toList.flatMap { (pid, evs) =>
      val positions = ordered(evs).flatMap { e =>
        e.eventType match
          case "POSITION_CHANGED" => dataNum(e, "toPosition").toList
          case "LAP_COMPLETED"    => dataNum(e, "position").toList
          case _                  => Nil
      }
      if positions.size < 2 then None
      else
        val first = positions.head.toInt
        val last = positions.last.toInt
        val improved = first - last
        if first >= 3 && (last <= 2 || improved >= 2) && improved > 0 then
          Some(JObj(Map(
            "type"     -> JStr("remontada"),
            "playerId" -> JStr(pid),
            "detected" -> JBool(true),
            "detail"   -> JStr(s"Jugador $pid remonto de posicion $first a $last")
          )))
        else None
    }

  // ---------- Combat ----------
  def combatStats(events: List[Event]): JObj =
    val attacks = events.filter(_.eventType == "PLAYER_ATTACKED")
    val totalDamage = attacks.flatMap(e => dataNum(e, "damage")).foldLeft(0.0)(_ + _)
    val avgDamage = if attacks.nonEmpty then totalDamage / attacks.size else 0.0
    val elims = events.filter(_.eventType == "PLAYER_ELIMINATED")
    // killer: data.killer, fallback playerId
    val killers: List[String] = elims.map(e => {
      val k = dataStr(e, "killer")
      if k.nonEmpty then k else e.playerId
    }).filter(k => k.nonEmpty && k != "world")
    val byKiller: Map[String, Int] = killers.groupBy(identity).map((k, v) => (k, v.size))
    val top: Option[String] = byKiller.toList.sortBy(-_._2).map(_._1).headOption
    val byWeapon: Map[String, Int] =
      elims.flatMap(e => { val w = dataStr(e, "weapon"); if w.nonEmpty then Some(w) else None })
        .groupBy(identity).map((k, v) => (k, v.size))
    JObj(Map(
      "totalAttacks"      -> JNum(attacks.size.toDouble),
      "totalDamage"       -> JNum(totalDamage),
      "averageDamage"     -> JNum(avgDamage),
      "totalEliminations" -> JNum(elims.size.toDouble),
      "topEliminator"     -> top.map(JStr(_)).getOrElse(JNull),
      "killsByWeapon"     -> JObj(byWeapon.map((k, v) => (k, JNum(v.toDouble)))),
      // Combinada: daño por eliminación (relaciona ATTACKED + ELIMINATED)
      "damagePerElimination" -> JNum(if elims.nonEmpty then totalDamage / elims.size else 0.0)
    ))

  def combatByPlayer(events: List[Event], players: List[String]): JObj =
    JObj(players.distinct.map { pid =>
      val elims = events.filter(_.eventType == "PLAYER_ELIMINATED")
      val kills = elims.count(e => dataStr(e, "killer") == pid)
      // Muertes: data.victim manda (el simulador emite playerId vacio); fallback a playerId solo si no hay victim
      val deaths = elims.count(e => {
        val v = dataStr(e, "victim")
        if v.nonEmpty then v == pid
        else e.playerId == pid && dataStr(e, "killer") != pid
      })
      val dmg = events.filter(e => e.eventType == "PLAYER_ATTACKED" && (e.playerId == pid || dataStr(e, "attacker") == pid))
        .flatMap(e => dataNum(e, "damage")).foldLeft(0.0)(_ + _)
      val atk = events.count(e => e.eventType == "PLAYER_ATTACKED" && (e.playerId == pid || dataStr(e, "attacker") == pid))
      val kd: Double = if deaths == 0 then kills.toDouble else kills.toDouble / deaths
      pid -> JObj(Map(
        "eliminations"    -> JNum(kills.toDouble),
        "timesEliminated" -> JNum(deaths.toDouble),
        "damageInflicted" -> JNum(dmg),
        "averageDamage"   -> JNum(if atk > 0 then dmg / atk else 0.0),
        "killDeathRatio"  -> JNum(kd)
      ))
    }.toMap)

  /** Racha: >= 3 eliminaciones como killer sin ser víctima en el medio (orden cronológico). */
  def detectStreaks(events: List[Event]): List[JObj] =
    val elims = ordered(events.filter(_.eventType == "PLAYER_ELIMINATED"))
    val killers = elims.map(e => { val k = dataStr(e, "killer"); if k.nonEmpty then k else e.playerId })
      .filter(k => k.nonEmpty && k != "world").distinct
    killers.flatMap { k =>
      val (maxStreak, _) = elims.foldLeft((0, 0)) { case ((mx, cur), e) =>
        val killer = { val x = dataStr(e, "killer"); if x.nonEmpty then x else e.playerId }
        val victim = dataStr(e, "victim")
        if killer == k then (mx.max(cur + 1), cur + 1)
        else if victim == k then (mx, 0) // lo eliminaron: se corta la racha
        else (mx, cur)
      }
      if maxStreak >= 3 then Some(JObj(Map(
        "type"     -> JStr("racha_eliminaciones"),
        "playerId" -> JStr(k),
        "detected" -> JBool(true),
        "detail"   -> JStr(s"Jugador $k logro racha de $maxStreak eliminaciones")
      )))
      else None
    }

  // ---------- Entrada principal ----------
  def analyze(req: MatchRequest): JObj =
    val evs = ordered(req.events)
    val players = evs.map(_.playerId).filter(_.nonEmpty).distinct
    val (stats, patterns) = req.gameId match
      case "racing" =>
        val s = racingStats(evs)
        val byP = racingByPlayer(evs)
        val merged = JObj(s.fields + ("byPlayer" -> byP))
        (merged, detectComebacks(evs))
      case "combat" =>
        val s = combatStats(evs)
        val byP = combatByPlayer(evs, players)
        val merged = JObj(s.fields + ("byPlayer" -> byP))
        (merged, detectStreaks(evs))
      case other =>
        (JObj(Map(
          "eventsCount"  -> JNum(evs.size.toDouble),
          "playersCount" -> JNum(players.size.toDouble),
          "note"         -> JStr(s"juego '$other' sin análisis especializado; estructura válida")
        )), Nil)
    JObj(Map(
      "status"     -> JStr("analyzed"),
      "matchId"    -> JStr(req.matchId),
      "gameId"     -> JStr(req.gameId),
      "statistics" -> stats,
      "patterns"   -> JArr(patterns)
    ))
end Analytics
