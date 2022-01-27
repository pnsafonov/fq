"$ fq -d kaitai -o source=@formats/\($n | gsub(".kst$";".ksy")) \"d,tovalue\" src/\(.data)",
( [ .asserts[]?
  | { key: .actual
    , value:
        ( .expected
        | if type == "string" then
            from_jq? // .
          end
        )
    }
  ]
| from_entries
)
