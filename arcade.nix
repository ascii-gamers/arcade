{ lib, buildGoModule }:

buildGoModule rec {
  pname = "arcade";
  version = "head";
  src = ./.;

  vendorSha256 = "sha256-zb+fU3qUPLXzgRLSDyTWrNjpDn0jiKrqSeej7yg808A=";
  checkPhase = false;

  meta = with lib; {
    description = "Multiplayer TRON using the Raft consensus algorithm";
    homepage = "https://github.com/ascii-gamers/arcade";
    license = licenses.mit;
    maintainers = with maintainers; [ siraben ];
  };
}
