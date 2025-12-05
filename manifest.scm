(use-modules (guix packages)
	     (gnu packages golang))

(packages->manifest (list go-1.25))
