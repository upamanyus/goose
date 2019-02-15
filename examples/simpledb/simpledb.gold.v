From RecoveryRefinement Require Import Database.CodeSetup.

Module Table.
  (* A Table provides access to an immutable copy of data on the filesystem, along
  with an index for fast random access. *)
  Record t := mk {
    Index: HashTable uint64;
    File: Fd;
  }.
End Table.

(* CreateTable creates a new, empty table. *)
Definition CreateTable (p:Path) : proc Table.t :=
  index <- Data.newHashTable uint64;
  f <- FS.create p;
  _ <- FS.close f;
  f2 <- FS.open p;
  Ret {| Table.Index := index;
         Table.File := f2; |}.

Module Entry.
  (* Entry represents a (key, value) pair. *)
  Record t := mk {
    Key: uint64;
    Value: slice.t byte;
  }.
End Entry.

(* DecodeUInt64 is a Decoder(uint64)

All decoders have the shape func(p []byte) (T, uint64)

The uint64 represents the number of bytes consumed; if 0, then decoding
failed, and the value of type T should be ignored. *)
Definition DecodeUInt64 (p:slice.t byte) : proc (uint64 * uint64) :=
  if compare (slice.length p) (fromNum 8) == Lt
  then Ret (0, 0)
  else
    n <- Data.uint64Get p;
    Ret (n, fromNum 8).

(* DecodeEntry is a Decoder(Entry) *)
Definition DecodeEntry (data:slice.t byte) : proc (Entry.t * uint64) :=
  let! (key, l1) <- DecodeUInt64 data;
  if l1 == 0
  then
    Ret ({| Entry.Key := 0;
            Entry.Value := slice.nil _; |}, 0)
  else
    let! (valueLen, l2) <- DecodeUInt64 (slice.skip l1 data);
    if l2 == 0
    then
      Ret ({| Entry.Key := 0;
              Entry.Value := slice.nil _; |}, 0)
    else
      let value := slice.subslice (l1 + l2) (l1 + l2 + valueLen) data in
      Ret ({| Entry.Key := key;
              Entry.Value := value; |}, l1 + l2 + valueLen).

Module lazyFileBuf.
  Record t := mk {
    offset: uint64;
    next: slice.t byte;
  }.
End lazyFileBuf.

(* readTableIndex parses a complete table on disk into a key->offset index *)
Definition readTableIndex (f:Fd) (index:HashTable uint64) : proc unit :=
  Loop (fun buf =>
        let! (e, l) <- DecodeEntry buf.(lazyFileBuf.next);
        if compare l 0 == Gt
        then
          _ <- Data.hashTableAlter index e.(Entry.Key) (fun _ => Some (fromNum 8 + buf.(lazyFileBuf.offset)));
          Continue {| lazyFileBuf.offset := buf.(lazyFileBuf.offset) + 1;
                      lazyFileBuf.next := slice.skip l buf.(lazyFileBuf.next); |}
        else
          p <- Base.sliceReadAt f buf.(lazyFileBuf.offset) 4096;
          if slice.length p == 0
          then LoopRet tt
          else
            newBuf <- Data.sliceAppendSlice buf.(lazyFileBuf.next) p;
            Continue {| lazyFileBuf.offset := buf.(lazyFileBuf.offset);
                        lazyFileBuf.next := newBuf; |}) {| lazyFileBuf.offset := 0;
           lazyFileBuf.next := slice.nil _; |}.

(* RecoverTable restores a table from disk on startup. *)
Definition RecoverTable (p:Path) : proc Table.t :=
  index <- Data.newHashTable uint64;
  f <- FS.open p;
  _ <- readTableIndex f index;
  Ret {| Table.Index := index;
         Table.File := f; |}.

(* CloseTable frees up the fd held by a table. *)
Definition CloseTable (t:Table.t) : proc unit :=
  FS.close t.(Table.File).

Definition ReadValue (f:Fd) (off:uint64) : proc (slice.t byte) :=
  buf <- Base.sliceReadAt f off 4096;
  totalBytes <- Data.uint64Get buf;
  let haveBytes := slice.length (slice.skip (fromNum 8) buf) in
  if compare haveBytes totalBytes == Lt
  then
    buf2 <- Base.sliceReadAt f (off + 4096) (totalBytes - haveBytes);
    newBuf <- Data.sliceAppendSlice buf buf2;
    Ret newBuf
  else Ret buf.

Definition TableRead (t:Table.t) (k:uint64) : proc (slice.t byte) :=
  let! (off, ok) <- Data.goHashTableLookup t.(Table.Index) k;
  if negb ok
  then Ret (slice.nil _)
  else
    p <- ReadValue t.(Table.File) off;
    Ret p.

Module bufFile.
  Record t := mk {
    file: Fd;
    buf: IORef (slice.t byte);
  }.
End bufFile.

Definition newBuf (f:Fd) : proc bufFile.t :=
  buf <- Data.newIORef (zeroValue (slice.t byte));
  Ret {| bufFile.file := f;
         bufFile.buf := buf; |}.

Definition bufFlush (f:bufFile.t) : proc unit :=
  buf <- Data.readIORef f.(bufFile.buf);
  if slice.length buf == 0
  then Ret ()
  else
    _ <- Base.sliceAppend f.(bufFile.file) buf;
    Data.writeIORef f.(bufFile.buf) (slice.nil _).

Definition bufAppend (f:bufFile.t) (p:slice.t byte) : proc unit :=
  buf <- Data.readIORef f.(bufFile.buf);
  buf2 <- Data.sliceAppendSlice buf p;
  Data.writeIORef f.(bufFile.buf) buf2.

Definition bufClose (f:bufFile.t) : proc unit :=
  _ <- bufFlush f;
  FS.close f.(bufFile.file).
