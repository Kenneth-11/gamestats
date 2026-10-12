package gamestats

import org.scalatest.flatspec.AnyFlatSpec
import org.scalatest.matchers.should.Matchers

class AnalyticsSpec extends AnyFlatSpec with Matchers:

  private def ev(id: String, ts: String, game: String, m: String, p: String, t: String, data: (String, J)*): Event =
    Event(id, ts, game, "1.0.0", m, p, t, "", JObj(data.toMap))

  "Racing" should "calcular vueltas, mejor y promedio (filter/map/foldLeft/reduceOption)" in {
    val events = List(
      ev("e1", "2026-03-15T14:00:00Z", "racing", "m1", "p1", "LAP_COMPLETED",
        "lapNumber" -> JNum(1), "lapTimeMs" -> JNum(1000), "averageSpeed" -> JNum(150), "position" -> JNum(2)),
      ev("e2", "2026-03-15T14:01:00Z", "racing", "m1", "p1", "LAP_COMPLETED",
        "lapNumber" -> JNum(2), "lapTimeMs" -> JNum(900), "averageSpeed" -> JNum(160), "position" -> JNum(1))
    )
    val s = Analytics.racingStats(events)
    s.fields("lapsCompleted") shouldBe JNum(2)
    s.fields("bestLapTime") shouldBe JNum(900)
    s.fields("averageLapTime") shouldBe JNum(950.0)
  }

  it should "agrupar por jugador (groupBy)" in {
    val events = List(
      ev("e1", "2026-03-15T14:00:00Z", "racing", "m1", "p1", "LAP_COMPLETED",
        "lapTimeMs" -> JNum(1000), "position" -> JNum(1)),
      ev("e2", "2026-03-15T14:01:00Z", "racing", "m1", "p2", "PENALTY_RECEIVED", "penaltyType" -> JStr("cut"))
    )
    val byP = Analytics.racingByPlayer(events)
    byP.fields.keySet should contain allOf ("p1", "p2")
  }

  it should "detectar remontada (secuencia POSITION_CHANGED)" in {
    val events = List(
      ev("e1", "2026-03-15T14:00:00Z", "racing", "m1", "p1", "POSITION_CHANGED",
        "fromPosition" -> JNum(5), "toPosition" -> JNum(3)),
      ev("e2", "2026-03-15T14:01:00Z", "racing", "m1", "p1", "POSITION_CHANGED",
        "fromPosition" -> JNum(3), "toPosition" -> JNum(1))
    )
    val pats = Analytics.detectComebacks(events)
    pats should have size 1
    pats.head.fields("playerId") shouldBe JStr("p1")
  }

  it should "no marcar remontada sin mejora" in {
    val events = List(
      ev("e1", "2026-03-15T14:00:00Z", "racing", "m1", "p1", "POSITION_CHANGED",
        "fromPosition" -> JNum(1), "toPosition" -> JNum(1))
    )
    Analytics.detectComebacks(events) shouldBe empty
  }

  "Combate" should "calcular daño total y top eliminador" in {
    val events = List(
      ev("a1", "2026-03-15T15:00:00Z", "combat", "m2", "p1", "PLAYER_ATTACKED",
        "attacker" -> JStr("p1"), "target" -> JStr("p2"), "damage" -> JNum(45), "weapon" -> JStr("rifle")),
      ev("a2", "2026-03-15T15:00:01Z", "combat", "m2", "p1", "PLAYER_ATTACKED",
        "attacker" -> JStr("p1"), "target" -> JStr("p2"), "damage" -> JNum(55), "weapon" -> JStr("rifle")),
      ev("k1", "2026-03-15T15:00:02Z", "combat", "m2", "p2", "PLAYER_ELIMINATED",
        "victim" -> JStr("p2"), "killer" -> JStr("p1"), "weapon" -> JStr("rifle"))
    )
    val s = Analytics.combatStats(events)
    s.fields("totalAttacks") shouldBe JNum(2)
    s.fields("totalDamage") shouldBe JNum(100)
    s.fields("averageDamage") shouldBe JNum(50.0)
    s.fields("topEliminator") shouldBe JStr("p1")
  }

  it should "detectar racha de 3 sin morir (foldLeft cronológico)" in {
    val events = (1 to 3).toList.map(i =>
      ev(s"e$i", s"2026-03-15T15:00:0${i}Z", "combat", "m1", "p1", "PLAYER_ELIMINATED",
        "victim" -> JStr(s"enemy$i"), "killer" -> JStr("p1"), "weapon" -> JStr("rifle")))
    val pats = Analytics.detectStreaks(events)
    pats should have size 1
  }

  it should "cortar la racha si el jugador muere en el medio" in {
    val events = List(
      ev("e1", "2026-03-15T15:00:00Z", "combat", "m1", "x", "PLAYER_ELIMINATED",
        "victim" -> JStr("a"), "killer" -> JStr("p1"), "weapon" -> JStr("rifle")),
      ev("e2", "2026-03-15T15:00:01Z", "combat", "m1", "x", "PLAYER_ELIMINATED",
        "victim" -> JStr("b"), "killer" -> JStr("p1"), "weapon" -> JStr("rifle")),
      ev("e3", "2026-03-15T15:00:02Z", "combat", "m1", "x", "PLAYER_ELIMINATED",
        "victim" -> JStr("p1"), "killer" -> JStr("other"), "weapon" -> JStr("rifle")),
      ev("e4", "2026-03-15T15:00:03Z", "combat", "m1", "x", "PLAYER_ELIMINATED",
        "victim" -> JStr("c"), "killer" -> JStr("p1"), "weapon" -> JStr("rifle"))
    )
    Analytics.detectStreaks(events) shouldBe empty
  }

  "Colecciones vacías" should "no romper (reduceOption)" in {
    val s = Analytics.racingStats(Nil)
    s.fields("lapsCompleted") shouldBe JNum(0)
    s.fields("bestLapTime") shouldBe JNull
  }

  "Juego desconocido" should "devolver nota sin romper" in {
    val req = MatchRequest("m9", "ajedrez", List(
      ev("e1", "2026-03-15T14:00:00Z", "ajedrez", "m9", "p1", "MOVE", "from" -> JStr("e2"))))
    val out = Analytics.analyze(req)
    out.fields("status") shouldBe JStr("analyzed")
  }

  "Json" should "parsear y renderizar ida y vuelta" in {
    val Right(v) = Json.parse("""{"a":1,"b":[true,null,"x"]}"""): @unchecked
    Json.render(v) should include("\"a\":1")
  }
end AnalyticsSpec
