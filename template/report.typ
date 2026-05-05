#import "lib.typ": (
  abbreviate-by-caps, code-block, get-var, header-border-color, lab-report, lab-section, summarize-name,
  table-border-width,
)
#import "@preview/elembic:1.1.1" as e

#let doc_title = sys.inputs.at("title", default: "Informe de Laboratorio")
#show: e.set_(code-block, lang: "python")

#let define(name, value) = {
  [#metadata((name: name, value: value)) <var_export>]
}

// Required vars: course_name, lab_title, lab_number, instructor_name, members
// Optional vars: year, presentation_date, course_abbr, member_abbr_list, sem_code, presentation_hour
// Anything else you can use for submission.js config

#define("course_name", "Ingeniería de Software")
#define("lab_title", "Título de la Práctica")
#define("lab_number", "01")
#define("instructor_name", "Nombre del Docente")
#define("members", (
  "Apellidos Apellidos Nombres Nombres",
  "Apellidos Apellidos Nombres Nombres",
  "Apellidos Apellidos Nombres Nombres",
))

#context {
  define("course_abbr", abbreviate-by-caps(get-var("course_name")))
  define("members_abbr_list", get-var("members").map(name => summarize-name(name)).join("_"))
  define("full_lab_number", numbering("001", int(get-var("lab_number"))))
}

#lab-report()[
  #set image(width: 78%)
  #set list(indent: 2pt)
  #show raw.where(block: false): it => box(
    inset: (x: 0.5pt),
  )[#it]

  #lab-section("RESULTADOS Y PRUEBAS")[
    #show heading: set text(weight: "bold")
    #set par(justify: true)

    = Ejercicio 1

    #lorem(3)

    #code-block("src/e1/placeholder.txt")
  ]

  #v(0.5em)

  #lab-section("CUESTIONARIO")[
    #show heading: set text(weight: "bold")
    #set par(justify: true)

    = Pregunta 1
  ]

  #v(0.5em)

  #lab-section("CONCLUSIONES")[
    #show heading: set text(weight: "bold")
    #set par(justify: true)

    = Conclusión 1
  ]

  #v(0.5em)

  #lab-section("REFERENCIAS")[
    #show heading: set text(weight: "bold")
    #bibliography("bibliography.bib", style: "ieee")
  ]
]
