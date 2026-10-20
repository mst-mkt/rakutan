{ lib, buildGoModule }:

buildGoModule {
  pname = "rakutan";
  version = "0.1.0";

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [
      ./go.mod
      ./go.sum
      ./main.go
      ./internal
    ];
  };

  vendorHash = "sha256-RQdyi1/s2AJSU9HlmexfrN9yXZOhPt9KH7qWbCQvAM8=";

  ldflags = [
    "-s"
    "-w"
  ];

  meta = {
    mainProgram = "rakutan";
    platforms = lib.platforms.unix;
  };
}
