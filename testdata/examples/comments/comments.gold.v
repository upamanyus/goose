(* autogenerated from github.com/tchajed/goose/testdata/examples/comments *)
From New.golang Require Import defn.

Section code.
Context `{ffi_syntax}.
Local Coercion Var' s: expr := Var s.

(* 0consts.go *)

Definition ONE : expr := #1.

Definition TWO : expr := #2.

(* 1doc.go *)

(* comments tests package comments, like this one

   it has multiple files *)

Definition Foo : go_type := structT [
  "a" :: boolT
].

Definition Foo__mset : list (string * val) := [
].

End code.
