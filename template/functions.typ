#let abbreviate-by-caps(word, separator: "") = {
  let chars = word.clusters()
  let caps = chars.filter(c => c == upper(c) and c != lower(c))
  caps.join(separator)
}

#let summarize-name(name, positions: (0, 2), separator: ",") = {
  let parts = name.split(" ")
  positions.map(pos => parts.at(pos)).join(separator)
}
