name := "gamestats-scala"
version := "1.0.0"
scalaVersion := "3.3.1"

// Servicio principal: solo stdlib (JDK HttpServer). Sin dependencias externas.
// Solo los tests usan scalatest.
libraryDependencies += "org.scalatest" %% "scalatest" % "3.2.17" % Test

Compile / mainClass := Some("gamestats.Main")

assembly / mainClass := Some("gamestats.Main")
assembly / assemblyJarName := "app.jar"
assembly / assemblyMergeStrategy := {
  case PathList("META-INF", _*) => MergeStrategy.discard
  case "reference.conf"          => MergeStrategy.concat
  case _                         => MergeStrategy.first
}
