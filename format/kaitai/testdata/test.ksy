meta:
  id: test
seq:
  - id: numbers1
    type: u2
  - id: numbers3
    type: s8
    repeat: expr
    repeat-expr: 10
  - id: test
    type: test
types:
  test:
    seq:
      - id: a
        type: u1
