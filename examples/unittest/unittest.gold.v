From RecoveryRefinement.Goose Require Import base.

Definition TypedLiteral  : proc uint64 :=
  Ret 3.

Definition LiteralCast  : proc uint64 :=
  let x := 2 in
  Ret (x + 2).

Definition CastInt (p:slice.t byte) : proc uint64 :=
  Ret (slice.length p).

Definition StringToByteSlice (s:string) : proc (slice.t byte) :=
  p <- Data.stringToBytes s;
  Ret p.

Definition ByteSliceToString (p:slice.t byte) : proc string :=
  s <- Data.bytesToString p;
  Ret s.

Definition UseSlice  : proc unit :=
  s <- Data.newSlice byte 1;
  s1 <- Data.sliceAppendSlice s s;
  FS.atomicCreate "file" s1.

Definition UseMap  : proc unit :=
  m <- Data.newMap (slice.t byte);
  _ <- Data.mapAlter m 1 (fun _ => Some (slice.nil _));
  let! (x, ok) <- Data.mapGet m 2;
  if ok
  then Ret tt
  else Data.mapAlter m 3 (fun _ => Some x).

Definition UsePtr  : proc unit :=
  p <- Data.newPtr uint64;
  _ <- Data.writePtr p 1;
  x <- Data.readPtr p;
  Data.writePtr p x.

Definition IterMapKeysAndValues (m:Map uint64) : proc uint64 :=
  sumPtr <- Data.newPtr uint64;
  _ <- Data.mapIter m (fun k v =>
    sum <- Data.readPtr sumPtr;
    Data.writePtr sumPtr (sum + k + v));
  sum <- Data.readPtr sumPtr;
  Ret sum.

Definition IterMapKeys (m:Map uint64) : proc (slice.t uint64) :=
  keysSlice <- Data.newSlice uint64 0;
  keysRef <- Data.newPtr (slice.t uint64);
  _ <- Data.writePtr keysRef keysSlice;
  _ <- Data.mapIter m (fun k _ =>
    keys <- Data.readPtr keysRef;
    newKeys <- Data.sliceAppend keys k;
    Data.writePtr keysRef newKeys);
  keys <- Data.readPtr keysRef;
  Ret keys.

Definition Empty  : proc unit :=
  Ret tt.

Definition EmptyReturn  : proc unit :=
  Ret tt.

Module allTheLiterals.
  Record t := mk {
    int: uint64;
    s: string;
    b: bool;
  }.
  Global Instance t_zero : HasGoZero t := mk (zeroValue _) (zeroValue _) (zeroValue _).
End allTheLiterals.

Definition normalLiterals  : proc allTheLiterals.t :=
  Ret {| allTheLiterals.int := 0;
         allTheLiterals.s := "foo";
         allTheLiterals.b := true; |}.

Definition specialLiterals  : proc allTheLiterals.t :=
  Ret {| allTheLiterals.int := 4096;
         allTheLiterals.s := "";
         allTheLiterals.b := false; |}.

Definition oddLiterals  : proc allTheLiterals.t :=
  Ret {| allTheLiterals.int := 5;
         allTheLiterals.s := "backquote string";
         allTheLiterals.b := false; |}.

Definition ReturnTwo (p:slice.t byte) : proc (uint64 * uint64) :=
  Ret (0, 0).

Definition ReturnTwoWrapper (data:slice.t byte) : proc (uint64 * uint64) :=
  let! (a, b) <- ReturnTwo data;
  Ret (a, b).

Definition DoSomeLocking (l:LockRef) : proc unit :=
  _ <- Data.lockAcquire l Writer;
  _ <- Data.lockRelease l Writer;
  _ <- Data.lockAcquire l Reader;
  _ <- Data.lockAcquire l Reader;
  _ <- Data.lockRelease l Reader;
  Data.lockRelease l Reader.

Definition MakeLock  : proc unit :=
  l <- Data.newLock;
  DoSomeLocking l.
