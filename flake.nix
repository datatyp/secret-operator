{
  description = "secret-operator";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      v = "v0.1.1";
      


      secretOperator = pkgs.buildGoModule rec {
        pname = "secret-operator";
        version = v;
        vendorHash = "sha256-fsAyRkS4VGhS3BEl4IaqN4d8LNxh9TGgoYZYR4qWFa4=";
        src = self;
      };

      dockerImage = pkgs.dockerTools.buildImage {
        name = "secret-operator";
        tag = secretOperator.version;
        config = {
          Cmd = [ "${secretOperator}/bin/secret-operator" ];
          Env = [ "SOURCE_NAMESPACE=kafka" ];
        };
      };

    in
    {
      defaultPackage.${system} = secretOperator;
      packages.${system}.dockerImage = dockerImage;


      defaultDevShell.${system} = pkgs.mkShell {
        buildInputs = [ secretOperator ];
        shellHook = ''
          export VERSION=${secretOperator.version}
      #   '';
      };
    };
}
