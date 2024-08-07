(* autogenerated from github.com/goose-lang/goose/testdata/examples/append_log *)
From New.golang Require Import defn.
From New.code Require github_com.goose_lang.goose.machine.disk.
From New.code Require github_com.tchajed.marshal.
From New.code Require sync.

From New Require Import disk_prelude.

(* Append-only, sequential, crash-safe log.

   The main interesting feature is that the log supports multi-block atomic
   appends, which are implemented by atomically updating an on-disk header with
   the number of valid blocks in the log. *)

Definition Log : go_type := structT [
  "m" :: ptrT;
  "sz" :: uint64T;
  "diskSz" :: uint64T
].

Definition Log__mkHdr : val :=
  rec: "Log__mkHdr" "log" <> :=
    exception_do (let: "log" := ref_ty ptrT "log" in
    let: "enc" := ref_ty marshal.Enc (zero_val marshal.Enc) in
    let: "$a0" := marshal.NewEnc disk.BlockSize in
    do:  "enc" <-[marshal.Enc] "$a0";;;
    do:  (marshal.Enc__PutInt (![marshal.Enc] "enc")) (![uint64T] (struct.field_ref Log "sz" (![ptrT] "log")));;;
    do:  (marshal.Enc__PutInt (![marshal.Enc] "enc")) (![uint64T] (struct.field_ref Log "diskSz" (![ptrT] "log")));;;
    return: ((marshal.Enc__Finish (![marshal.Enc] "enc")) #());;;
    do:  #()).

Definition Log__writeHdr : val :=
  rec: "Log__writeHdr" "log" <> :=
    exception_do (let: "log" := ref_ty ptrT "log" in
    do:  disk.Write #0 ((Log__mkHdr (![ptrT] "log")) #());;;
    do:  #()).

Definition Init : val :=
  rec: "Init" "diskSz" :=
    exception_do (let: "diskSz" := ref_ty uint64T "diskSz" in
    (if: (![uint64T] "diskSz") < #1
    then
      return: (ref_ty Log (struct.make Log [{
         "m" ::= ref_ty sync.Mutex (zero_val sync.Mutex);
         "sz" ::= #0;
         "diskSz" ::= #0
       }]), #false);;;
      do:  #()
    else do:  #());;;
    let: "log" := ref_ty ptrT (zero_val ptrT) in
    let: "$a0" := ref_ty Log (struct.make Log [{
      "m" ::= ref_ty sync.Mutex (zero_val sync.Mutex);
      "sz" ::= #0;
      "diskSz" ::= ![uint64T] "diskSz"
    }]) in
    do:  "log" <-[ptrT] "$a0";;;
    do:  (Log__writeHdr (![ptrT] "log")) #();;;
    return: (![ptrT] "log", #true);;;
    do:  #()).

Definition Open : val :=
  rec: "Open" <> :=
    exception_do (let: "hdr" := ref_ty (sliceT byteT) (zero_val (sliceT byteT)) in
    let: "$a0" := disk.Read #0 in
    do:  "hdr" <-[sliceT byteT] "$a0";;;
    let: "dec" := ref_ty marshal.Dec (zero_val marshal.Dec) in
    let: "$a0" := marshal.NewDec (![sliceT byteT] "hdr") in
    do:  "dec" <-[marshal.Dec] "$a0";;;
    let: "sz" := ref_ty uint64T (zero_val uint64T) in
    let: "$a0" := (marshal.Dec__GetInt (![marshal.Dec] "dec")) #() in
    do:  "sz" <-[uint64T] "$a0";;;
    let: "diskSz" := ref_ty uint64T (zero_val uint64T) in
    let: "$a0" := (marshal.Dec__GetInt (![marshal.Dec] "dec")) #() in
    do:  "diskSz" <-[uint64T] "$a0";;;
    return: (ref_ty Log (struct.make Log [{
       "m" ::= ref_ty sync.Mutex (zero_val sync.Mutex);
       "sz" ::= ![uint64T] "sz";
       "diskSz" ::= ![uint64T] "diskSz"
     }]));;;
    do:  #()).

Definition Log__get : val :=
  rec: "Log__get" "log" "i" :=
    exception_do (let: "log" := ref_ty ptrT "log" in
    let: "i" := ref_ty uint64T "i" in
    let: "sz" := ref_ty uint64T (zero_val uint64T) in
    let: "$a0" := ![uint64T] (struct.field_ref Log "sz" (![ptrT] "log")) in
    do:  "sz" <-[uint64T] "$a0";;;
    (if: (![uint64T] "i") < (![uint64T] "sz")
    then
      return: (disk.Read (#1 + (![uint64T] "i")), #true);;;
      do:  #()
    else do:  #());;;
    return: (slice.nil, #false);;;
    do:  #()).

Definition Log__Get : val :=
  rec: "Log__Get" "log" "i" :=
    exception_do (let: "log" := ref_ty ptrT "log" in
    let: "i" := ref_ty uint64T "i" in
    do:  (sync.Mutex__Lock (![ptrT] (struct.field_ref Log "m" (![ptrT] "log")))) #();;;
    let: "b" := ref_ty boolT (zero_val boolT) in
    let: "v" := ref_ty (sliceT byteT) (zero_val (sliceT byteT)) in
    let: ("$a0", "$a1") := (Log__get (![ptrT] "log")) (![uint64T] "i") in
    do:  "b" <-[boolT] "$a1";;;
    do:  "v" <-[sliceT byteT] "$a0";;;
    do:  (sync.Mutex__Unlock (![ptrT] (struct.field_ref Log "m" (![ptrT] "log")))) #();;;
    return: (![sliceT byteT] "v", ![boolT] "b");;;
    do:  #()).

Definition writeAll : val :=
  rec: "writeAll" "bks" "off" :=
    exception_do (let: "off" := ref_ty uint64T "off" in
    let: "bks" := ref_ty (sliceT (sliceT byteT)) "bks" in
    do:  let: "$range" := ![sliceT (sliceT byteT)] "bks" in
    slice.for_range (sliceT byteT) "$range" (λ: "i" "bk",
      let: "i" := ref_ty uint64T "i" in
      let: "bk" := ref_ty (sliceT byteT) "bk" in
      do:  disk.Write ((![uint64T] "off") + (![intT] "i")) (![sliceT byteT] "bk");;;
      do:  #());;;
    do:  #()).

Definition Log__append : val :=
  rec: "Log__append" "log" "bks" :=
    exception_do (let: "log" := ref_ty ptrT "log" in
    let: "bks" := ref_ty (sliceT (sliceT byteT)) "bks" in
    let: "sz" := ref_ty uint64T (zero_val uint64T) in
    let: "$a0" := ![uint64T] (struct.field_ref Log "sz" (![ptrT] "log")) in
    do:  "sz" <-[uint64T] "$a0";;;
    (if: (slice.len (![sliceT (sliceT byteT)] "bks")) ≥ (((![uint64T] (struct.field_ref Log "diskSz" (![ptrT] "log"))) - #1) - (![uint64T] "sz"))
    then
      return: (#false);;;
      do:  #()
    else do:  #());;;
    do:  writeAll (![sliceT (sliceT byteT)] "bks") (#1 + (![uint64T] "sz"));;;
    do:  (struct.field_ref Log "sz" (![ptrT] "log")) <-[uint64T] ((![uint64T] (struct.field_ref Log "sz" (![ptrT] "log"))) + (slice.len (![sliceT (sliceT byteT)] "bks")));;;
    do:  (Log__writeHdr (![ptrT] "log")) #();;;
    return: (#true);;;
    do:  #()).

Definition Log__Append : val :=
  rec: "Log__Append" "log" "bks" :=
    exception_do (let: "log" := ref_ty ptrT "log" in
    let: "bks" := ref_ty (sliceT (sliceT byteT)) "bks" in
    do:  (sync.Mutex__Lock (![ptrT] (struct.field_ref Log "m" (![ptrT] "log")))) #();;;
    let: "b" := ref_ty boolT (zero_val boolT) in
    let: "$a0" := (Log__append (![ptrT] "log")) (![sliceT (sliceT byteT)] "bks") in
    do:  "b" <-[boolT] "$a0";;;
    do:  (sync.Mutex__Unlock (![ptrT] (struct.field_ref Log "m" (![ptrT] "log")))) #();;;
    return: (![boolT] "b");;;
    do:  #()).

Definition Log__reset : val :=
  rec: "Log__reset" "log" <> :=
    exception_do (let: "log" := ref_ty ptrT "log" in
    let: "$a0" := #0 in
    do:  (struct.field_ref Log "sz" (![ptrT] "log")) <-[uint64T] "$a0";;;
    do:  (Log__writeHdr (![ptrT] "log")) #();;;
    do:  #()).

Definition Log__Reset : val :=
  rec: "Log__Reset" "log" <> :=
    exception_do (let: "log" := ref_ty ptrT "log" in
    do:  (sync.Mutex__Lock (![ptrT] (struct.field_ref Log "m" (![ptrT] "log")))) #();;;
    do:  (Log__reset (![ptrT] "log")) #();;;
    do:  (sync.Mutex__Unlock (![ptrT] (struct.field_ref Log "m" (![ptrT] "log")))) #();;;
    do:  #()).
