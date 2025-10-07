load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_dependencies():
    go_repository(
        name = "com_github_gorilla_mux",
        importpath = "github.com/gorilla/mux",
        sum = "h1:TuBL49tXwgrFYWhqrNgrUNEY92u81SPhu7sTdzQEiWY=",
        version = "v1.8.1",
    )
    go_repository(
        name = "com_github_gorilla_websocket",
        importpath = "github.com/gorilla/websocket",
        sum = "h1:gmztn0JnHVt9JZquRuzLw3g4wouNVzKL15iLr/zn/QY=",
        version = "v1.5.1",
    )
