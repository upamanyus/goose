(* autogenerated from github.com/tchajed/goose/internal/examples/async *)
From Perennial.goose_lang Require Import prelude.
From Goose Require github_dot_com.tchajed.goose.machine.async__disk.

From Perennial.goose_lang Require Import ffi.async_disk_prelude.

(* async just uses the async disk FFI *)

Definition TakesDisk: val :=
  rec: "TakesDisk" "d" :=
    #().

Definition UseDisk: val :=
  rec: "UseDisk" "d" :=
    let: "v" := NewSlice byteT #4096 in
    disk.Write #0 "v";;
    disk.Barrier #();;
    #().
