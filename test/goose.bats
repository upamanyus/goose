setup_file() {
    # get the containing directory of this file
    # use $BATS_TEST_FILENAME instead of ${BASH_SOURCE[0]} or $0,
    # as those will point to the bats executable's location or the preprocessed file respectively
    test_file_dir="$( cd "$( dirname "$BATS_TEST_FILENAME" )" >/dev/null 2>&1 && pwd )"
    export GOOSE="$test_file_dir/.."
    cd "$GOOSE" || exit 1
    go build -o "$GOOSE/testdata/goose" ./cmd/goose
    export PATH="$GOOSE/testdata:$PATH"
    export TEST_DIR="$GOOSE/testdata/goose-tests"
    cd "$TEST_DIR" || exit 1
    # goose output should be emitted here
    export OUT="Goose/example_dot_com/goose_dash_demo"
}

setup() {
    load 'test_helper/bats-support/load'
    load 'test_helper/bats-assert/load'
}

teardown() {
    rm -rf "$TEST_DIR/Goose"
}

teardown_file() {
    rm "$GOOSE/testdata/goose"
}

# assert_file_exists and assert_file_not_exist were inspired by
# https://github.com/ztombol/bats-file

# assert file exists, with bats decoration on failure
assert_file_exists() {
  local -r file="$1"
  if [[ ! -e "$file" ]]; then
    batslib_print_kv_single 4 'path' "$file" \
      | batslib_decorate 'file does not exist' \
      | fail
  fi
}

# assert file does not exist, with bats decoration on failure
assert_file_not_exist() {
  local -r file="$1"
  if [[ -e "$file" ]]; then
    batslib_print_kv_single 4 'path' "$file" \
      | batslib_decorate 'file exists, but it was expected to be absent' \
      | fail
  fi
}

@test "goose current directory" {
    goose -out Goose
    run cat "$OUT"/m.v
    assert_output --partial "From Goose Require github_dot_com.tchajed.marshal."
    assert_output --partial "Section code."
}

@test "goose ." {
    goose -out Goose .
    assert_file_exists "$OUT"/m.v
}

@test "goose with multiple patterns" {
    goose -out Goose . ./use_disk ./use_grove
    assert_file_exists "$OUT"/m.v
    assert_file_exists "$OUT"/m/use__disk.v
    assert_file_exists "$OUT"/m/use__grove.v
}

@test "goose grove_ffi" {
    goose -out Goose ./use_grove
    run cat "$OUT"/m/use__grove.v
    assert_output --partial "Import ffi.grove_prelude"
}

@test "goose bad path" {
    run goose -out Goose ./not_a_thing
    assert_failure
    assert_output --partial "could not load package"
}

@test "goose with one error" {
    run goose -out Goose ./use_disk ./errors/not_goose
    # fails, but does output correct file
    assert_failure
    assert_file_exists "$OUT"/m/use__disk.v
    assert_file_not_exist "$OUT"/m/errors/not__goose.v
    assert_file_not_exist "$OUT"/m.v
}

@test "goose with build tag to suppress bad code" {
    goose -out Goose ./errors/build_tag
    run cat "$OUT"/m/errors/build__tag.v
    assert_output --partial "Definition Foo"
    refute_output --partial "WontTranslate"
}

@test "goose on ./..." {
    run goose -out Goose ./...
    assert_failure
    assert_file_exists "$OUT"/m.v
    assert_file_exists "$OUT"/m/use__disk.v
    assert_file_exists "$OUT"/m/errors/build__tag.v
}

@test "goose on external package" {
    goose -out Goose github.com/tchajed/marshal
    run cat Goose/github_dot_com/tchajed/marshal.v
    assert_output --partial "NewEnc"
}

@test "goose using -dir" {
    # run this test outside of the correct go.mod module
    cd
    goose -out "$TEST_DIR/Goose" -dir "$GOOSE/testdata/goose-tests"
    cd "$TEST_DIR"
    assert_file_exists "$OUT"/m.v
}

@test "goose local path" {
    goose -out Goose example.com/goose-demo/m
    assert_file_exists "$OUT"/m.v
}

@test "goose local path with subdir" {
    # use a sub-dir
    goose -out Goose -dir "use_disk" example.com/goose-demo/m
    assert_file_exists "$OUT"/m.v
    assert_file_not_exist "$OUT"/m/use_disk.v
}

@test "goose -ignore-errors" {
    run goose -out Goose -ignore-errors ./errors/not_goose
    # even -ignore-errors sets an error status
    assert_failure
    run cat "$OUT"/m/errors/not__goose.v
    # the top-level comment and other definitions should translate
    assert_output --partial "not_goose has code"
    assert_output --partial "Definition Number"
    refute_output --partial "Bad"
    assert_output --partial "Definition Ok"
}

@test "goose after change" {
  run goose -out Goose
  sed -i~ 's/UseMarshal/ExampleFunc/' m.go
  run goose -out Goose
  run cat "$OUT"/m.v
  assert_output --partial "ExampleFunc"
  sed -i~ 's/ExampleFunc/UseMarshal/' m.go
}
