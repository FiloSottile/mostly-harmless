import llm
import tempfile
import subprocess
import sys


@llm.hookimpl
def register_fragment_loaders(register):
    register("go", go_loader)


def go_loader(argument: str) -> llm.Fragment:
    return llm.Fragment(
        go_doc(argument),
        source=f"https://pkg.go.dev/{argument}",
    )


def go_doc(argument: str) -> str:
    package = argument.split("@")[0] if "@" in argument else argument
    with tempfile.TemporaryDirectory() as tmpdir:
        run = lambda cmd: subprocess.run(
            cmd, cwd=tmpdir, capture_output=True, check=True, text=True
        )
        try:
            run(["go", "mod", "init", "llm_fragments_go"])
            run(["go", "get", argument])
            result = run(["go", "doc", "-all", package])
            return result.stdout
        except subprocess.CalledProcessError as e:
            print(f"$ {' '.join(e.cmd)}", file=sys.stderr)
            print(e.stderr, file=sys.stderr)
            raise
