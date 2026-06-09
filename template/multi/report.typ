#import "/lib.typ": code-block, code-block-config, lab-report

#show: lab-report.with(
  course_name: "Ingeniería de Software",
  lab_title: "Título de la Práctica",
  lab_number: "01",
  instructor_name: "Nombre del Docente",
  members: (
    "Apellidos1 Apellidos1 Nombres1 Nombres1",
    "Apellidos2 Apellidos2 Nombres2 Nombres2",
    "Apellidos3 Apellidos3 Nombres3 Nombres3",
  ),
)

// Configure components
#code-block-config(lang: "python", prefix: "#")

#set image(width: 78%)
#set list(indent: 2pt)
#show raw.where(block: false): it => box(inset: (x: 0.5pt))[#it]

#include "sections/1-resultados.typ"
#v(0.5em)
#include "sections/2-cuestionario.typ"
#v(0.5em)
#include "sections/3-conclusiones.typ"
#v(0.5em)
#include "sections/4-referencias.typ"
