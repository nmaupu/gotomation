;; Eval Buffer with `M-x eval-buffer' to register the newly created template.

(dap-register-debug-template
  "Gotomation"
  (list :type "go"
        :request "launch"
        :name "Gotomation"
        :mode "debug"
        :program (projectile-project-root)
        :args "--config /home/nmaupu/work/tmp/gotomation-test.yaml -t eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiI0ZDIzOTAyMDUzYTk0NDI3YjEzNWRmNTdlYmIxMzYwNiIsImlhdCI6MTczMzU4MTk3MSwiZXhwIjoyMDQ4OTQxOTcxfQ.f7B0urtszF8ZpuA5WlEN_b8Z8kYkvdtiI5VLLbj07J8"
        :env nil))
