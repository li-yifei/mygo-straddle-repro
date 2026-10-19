// Usage: swift hit.swift X1 Y1 [X2 Y2 ...]   (global top-left screen points, as MyGo/CGWindow use)
import CoreGraphics
// Front-most on-screen, layer-0 window containing each point.
let list = CGWindowListCopyWindowInfo([.optionOnScreenOnly], kCGNullWindowID) as! [[String: Any]]
var i = 1
while i + 1 < CommandLine.arguments.count {
  let p = CGPoint(x: Double(CommandLine.arguments[i])!, y: Double(CommandLine.arguments[i+1])!)
  for w in list where (w["kCGWindowLayer"] as? Int ?? 0) == 0 {
    let b = w["kCGWindowBounds"] as! [String: Any]
    let r = CGRect(x: b["X"] as! Double, y: b["Y"] as! Double, width: b["Width"] as! Double, height: b["Height"] as! Double)
    if r.contains(p) { print("point (\(Int(p.x)),\(Int(p.y))) -> \(w["kCGWindowOwnerName"] as? String ?? "") / \(w["kCGWindowName"] as? String ?? "")"); break }
  }
  i += 2
}
