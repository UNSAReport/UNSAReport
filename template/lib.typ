#let table-border-width = 0.5pt
#let header-border-color = rgb("#808080")
#let tb-header-bg-color = rgb("#C8310E")
#let code-bg-color = rgb("#F1F3F4")

#let define(name, value) = {
  [#metadata((name: name, value: value)) <var_export>]
}

#let get-var(name, default: none) = {
  let vars = query(<var_export>)
  let match = vars.find(item => item.value.at("name") == name)
  if match == none {
    if default == none {
      panic("Missing exported var: " + name)
    }
    return default
  }
  match.value.at("value")
}

#let abbreviate-by-caps(word, separator: "") = {
  let chars = word.clusters()
  let caps = chars.filter(c => c == upper(c) and c != lower(c))
  caps.join(separator)
}

#let summarize-name(name, positions: (0, 2), separator: ",") = {
  let parts = name.split(" ")
  positions.map(pos => parts.at(pos)).join(separator)
}

// Configuration states
#let code-block-state = state(
  "code-block-config",
  (
    prefix: "//",
    lang: "text",
    fill: code-bg-color,
    breakable: true,
    width: 100%,
    inset: 1em,
    radius: 8pt,
    spacing: 0.65em,
    clip: false,
    text-size: 7pt,
  ),
)

#let code-block-config(..args) = {
  code-block-state.update(old => {
    let new = old
    for (key, value) in args.named() {
      new.insert(key, value)
    }
    new
  })
}

#let lab-section-state = state(
  "lab-section-config",
  (
    align-mode: left + top,
    stroke: black + 1pt,
    inset: 0.5em,
    header-fill: tb-header-bg-color,
  ),
)

#let lab-section-config(..args) = {
  lab-section-state.update(old => {
    let new = old
    for (key, value) in args.named() {
      new.insert(key, value)
    }
    new
  })
}

#let extract-named-snippet(source, file, snippet-name, prefix) = {
  let lines = source.split("\n")
  let start-marker = prefix + " START-SNIPPET," + snippet-name
  let end-marker = prefix + " END-SNIPPET"
  let result = lines.fold((false, false, ()), (acc, line) => {
    let found-start = acc.at(0)
    let found-end = acc.at(1)
    let captured = acc.at(2)

    if found-end {
      acc
    } else if not found-start and line.trim() == start-marker {
      (true, false, ())
    } else if found-start and line.trim() == end-marker {
      (true, true, captured)
    } else if found-start {
      (true, false, captured + (line,))
    } else {
      acc
    }
  })

  if result.at(0) and result.at(1) {
    result.at(2).join("\n")
  } else {
    panic("Snippet '" + snippet-name + "' not found or not closed in file: " + file)
  }
}

#let code-block(
  file: none,
  snippet: none,
  prefix: none,
  lang: none,
  fill: none,
  breakable: none,
  width: none,
  inset: none,
  radius: none,
  spacing: none,
  clip: none,
  text-size: none,
) = context {
  let config = code-block-state.get()

  let p_prefix = if prefix != none { prefix } else { config.prefix }
  let p_lang = if lang != none { lang } else { config.lang }
  let p_fill = if fill != none { fill } else { config.fill }
  let p_breakable = if breakable != none { breakable } else { config.breakable }
  let p_width = if width != none { width } else { config.width }
  let p_inset = if inset != none { inset } else { config.inset }
  let p_radius = if radius != none { radius } else { config.radius }
  let p_spacing = if spacing != none { spacing } else { config.spacing }
  let p_clip = if clip != none { clip } else { config.clip }
  let p_text_size = if text-size != none { text-size } else { config.text-size }

  let source = read(file)
  let code = if snippet == none {
    source
  } else {
    extract-named-snippet(source, file, snippet, p_prefix)
  }

  block(
    fill: p_fill,
    breakable: p_breakable,
    width: p_width,
    inset: p_inset,
    radius: p_radius,
    spacing: p_spacing,
    clip: p_clip,
  )[
    #set text(size: p_text_size)
    #set par(justify: false)
    #raw(code, lang: p_lang, block: true)
  ]
}

#let lab-section(
  title: [],
  align-mode: none,
  stroke: none,
  inset: none,
  header-fill: none,
  body,
) = context {
  let config = lab-section-state.get()

  let p_align_mode = if align-mode != none { align-mode } else { config.align-mode }
  let p_stroke = if stroke != none { stroke } else { config.stroke }
  let p_inset = if inset != none { inset } else { config.inset }
  let p_header_fill = if header-fill != none { header-fill } else { config.header-fill }

  grid(
    align: p_align_mode,
    stroke: p_stroke,
    inset: p_inset,
    columns: 1fr,
    grid.header(
      repeat: false,
      [#grid.cell(fill: p_header_fill)[
        #set text(size: 11pt, weight: "bold", fill: white)
        #align(center)[#title]
      ]],
    ),
    [
      #set text(size: 8.5pt)
      #body
    ],
  )
}

#let page-header() = block(
  width: 100%,
  inset: (bottom: 1em),
)[
  #table(
    align: center + horizon,
    stroke: table-border-width + header-border-color,
    columns: (1fr, 2fr, 1fr),
    align(horizon)[#image("img/fixed/epis.png", width: 95%)],
    table.cell(align: center + horizon)[
      #set text(size: 7.5pt, weight: "bold")
      UNIVERSIDAD NACIONAL DE SAN AGUSTÍN \
      FACULTAD DE INGENIERÍA DE PRODUCCIÓN Y SERVICIOS \
      ESCUELA PROFESIONAL DE INGENIERÍA DE SISTEMAS
    ],
    align(horizon)[#image("img/fixed/abet.png", width: 97%)],
    table.cell(colspan: 3)[
      #set text(size: 7pt)
      #text(weight: "bold")[Formato: ]
      Guía de Práctica de Laboratorio / Talleres / Centros de Simulación
    ],
    table.cell[
      #set text(size: 7pt, weight: "bold")
      Aprobación: 2022/03/01
    ],
    table.cell[
      #set text(size: 7pt, weight: "bold")
      Código: GUIA-PRLE-001
    ],
    context table.cell(align: right + horizon)[
      #set text(size: 7pt, weight: "bold")
      Página: #counter(page).display("1")
    ],
  )
]

#let basic-info-table(
  course-name,
  lab-title,
  lab-number,
  year,
  sem-code,
  presentation-date,
  presentation-hour,
  member-list,
  instructor-name,
) = [
  #show table.cell: set text(size: 8.5pt)
  #table(
    align: left + horizon,
    stroke: black + 1pt,
    inset: 0.5em,
    columns: (1fr, 1fr, 1fr, 1fr, 1fr, 1fr),
    table.cell(colspan: 6, fill: tb-header-bg-color, align: center + horizon)[
      #set text(size: 11pt, weight: "bold", fill: white)
      INFORMACIÓN BÁSICA
    ],
    [#text(weight: "bold")[ASIGNATURA:]],
    table.cell(colspan: 5)[#course-name],
    [#text(weight: "bold")[TÍTULO DE LA PRÁCTICA:]],
    table.cell(colspan: 5)[#lab-title],
    [#text(weight: "bold")[NÚMERO DE LA PRÁCTICA:]],
    [#lab-number],
    [#text(weight: "bold")[AÑO LECTIVO:]],
    [#year],
    [#text(weight: "bold")[NRO. SEMESTRE:]],
    [#sem-code],
    [#text(weight: "bold")[FECHA DE PRESENTACIÓN:]],
    [#presentation-date],
    [#text(weight: "bold")[HORA DE PRESENTACIÓN:]],
    table.cell(colspan: 3)[#presentation-hour],
    table.cell(colspan: 4)[
      #text(weight: "bold")[INTEGRANTE(s):] \
      #for member in member-list {
        [
          - #member
        ]
      }
    ],
    [#text(weight: "bold")[NOTA (0 - 20):]],
    [Nota colocada por el docente],
    table.cell(colspan: 6)[
      #text(weight: "bold")[DOCENTE: ] \
      #instructor-name
    ],
  )
]

#let lab-report(
  course_name: none,
  lab_title: none,
  lab_number: none,
  instructor_name: none,
  members: (),
  year: none,
  presentation_date: none,
  sem_code: none,
  presentation_hour: "11:59:00",
  ..custom_vars,
  body,
) = {
  // Export metadata for CLI
  define("course_name", course_name)
  define("lab_title", lab_title)
  define("lab_number", lab_number)
  define("instructor_name", instructor_name)
  define("members", members)

  let gen-time = datetime.today()
  let resolved-year = if year != none { year } else { gen-time.year() }
  let resolved-presentation-date = if presentation_date != none {
    presentation_date
  } else {
    gen-time.display("[day]/[month]/[year]")
  }
  let resolved-sem-code = if sem_code != none {
    sem_code
  } else {
    if gen-time.month() < 8 { "A" } else { "B" }
  }

  define("year", resolved-year)
  define("presentation_date", resolved-presentation-date)
  define("sem_code", resolved-sem-code)
  define("presentation_hour", presentation_hour)

  // Custom vars export
  for (name, value) in custom_vars.named() {
    define(name, value)
  }

  // Derived vars export
  let course_abbr = abbreviate-by-caps(course_name)
  let shortnames_chain = members.map(name => summarize-name(name)).join("_")
  let surnames_chain = members
    .map(name => summarize-name(name, positions: (0,), separator: ""))
    .join("-")
  let wide_lab_number = numbering("001", int(lab_number))

  define("course_abbr", course_abbr)
  define("shortnames_chain", shortnames_chain)
  define("surnames_chain", surnames_chain)
  define("wide_lab_number", wide_lab_number)

  // Layout
  set text(font: "Lato")
  show heading.where(level: 1): set text(size: 10pt)
  show heading.where(level: 2): set text(size: 9pt)
  set list(indent: 1em, marker: "-")
  set enum(numbering: "1.")
  set image(width: 90%)
  set figure(supplement: [Figura])
  show image: set align(center)

  set page(
    paper: "a4",
    margin: (
      top: 6cm,
      bottom: 2.54cm,
      left: 1.9cm,
      right: 1.9cm,
    ),
    header: page-header(),
    header-ascent: 5%,
  )

  align(center)[#text(size: 13pt, weight: "bold")[INFORME DE LABORATORIO]]

  basic-info-table(
    course_name,
    lab_title,
    lab_number,
    resolved-year,
    resolved-sem-code,
    resolved-presentation-date,
    presentation_hour,
    members,
    instructor_name,
  )

  body
}
