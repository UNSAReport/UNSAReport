#import "@preview/elembic:1.1.1" as e

#let table-border-width = 0.5pt
#let header-border-color = rgb("#808080")
#let tb-header-bg-color = rgb("#C8310E")
#let code-bg-color = rgb("#F1F3F4")

#let define(name, value) = {
  [#metadata((name: name, value: value)) <var_export>]
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

#let code-block = e.element.declare(
  "code-block",
  prefix: "@christianmz565/lab-report,v3",
  doc: "Displays source code from a file in a formatted block.",
  fields: (
    e.field("file", str, required: true),
    e.field("snippet", e.types.option(str), default: none),
    e.field("prefix", str, default: "//"),
    e.field("lang", str, default: "text"),
    e.field("fill", e.types.option(e.types.paint), default: code-bg-color),
    e.field("breakable", bool, default: true),
    e.field("width", e.types.any, default: 100%),
    e.field("inset", e.types.any, default: 1em),
    e.field("radius", e.types.any, default: 8pt),
    e.field("spacing", e.types.any, default: 0.65em),
    e.field("clip", bool, default: false),
    e.field("text-size", e.types.any, default: 7pt),
  ),
  display: it => {
    let source = read(it.file)
    let snippet-name = it.at("snippet")
    let code = if snippet-name == none {
      source
    } else {
      extract-named-snippet(source, it.file, snippet-name, it.at("prefix"))
    }

    block(
      fill: it.fill,
      breakable: it.breakable,
      width: it.width,
      inset: it.inset,
      radius: it.radius,
      spacing: it.spacing,
      clip: it.clip,
    )[
      #set text(size: it.at("text-size"))
      #set par(justify: false)
      #raw(code, lang: it.lang, block: true)
    ]
  },
)

#let lab-section = e.element.declare(
  "lab-section",
  prefix: "@christianmz565/lab-report,v3",
  doc: "Displays a report section with a header bar.",
  fields: (
    e.field("title", content, required: true),
    e.field("body", content, required: true),
    e.field("align-mode", alignment, default: left + top),
    e.field("stroke", e.types.any, default: black + 1pt),
    e.field("inset", e.types.any, default: 0.5em),
    e.field("header-fill", e.types.option(e.types.paint), default: tb-header-bg-color),
  ),
  display: it => grid(
    align: it.at("align-mode"),
    stroke: it.stroke,
    inset: it.inset,
    columns: 1fr,
    grid.header(
      repeat: false,
      [#grid.cell(
        fill: it.at("header-fill"),
      )[
        #set text(size: 11pt, weight: "bold", fill: white)
        #align(center)[#it.title]
      ]],
    ),
    [
      #set text(size: 8.5pt)
      #it.body
    ],
  ),
)

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

#let lab-report = e.element.declare(
  "lab-report",
  prefix: "@christianmz565/lab-report,v3",
  doc: "Main layout and metadata wrapper for EPIS lab reports.",
  fields: (
    e.field("body", content, required: true),
  ),
  display: it => context {
    let course-name = get-var("course_name")
    let lab-title = get-var("lab_title")
    let lab-number = get-var("lab_number")
    let instructor-name = get-var("instructor_name")
    let member-list = get-var("members")

    let gen-time = datetime.today()
    let resolved-year = get-var("year", default: gen-time.year())
    let resolved-presentation-date = get-var("presentation_date", default: gen-time.display("[day]/[month]/[year]"))
    let sem-code = get-var("sem_code", default: if gen-time.month() < 8 { "A" } else { "B" })
    let presentation-hour = get-var("presentation_hour", default: "11:59:00")

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
      course-name,
      lab-title,
      lab-number,
      resolved-year,
      sem-code,
      resolved-presentation-date,
      presentation-hour,
      member-list,
      instructor-name,
    )

    it.body
  },
)
