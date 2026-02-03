;;; pi-def-mode.el --- Major mode for pi CLI definition files (.def)

(defvar pi-def-highlights
  '(("^#.*" . font-lock-comment-face)
    ("\\b\\(global\\|cmd\\|topic\\)\\b" . font-lock-keyword-face)
    ("\\b\\(flag\\|arg\\|example\\|text\\)\\b" . font-lock-builtin-face)
    ("\\b\\(bool\\|string\\)\\b" . font-lock-type-face)
    ("cmd\\s-+\\([a-zA-Z0-9_/ ]+\\)\)" 1 font-lock-function-name-face)
    ("\\(?:flag\\|arg\\)\\s-+\\([a-zA-Z0-9_-]+\\)" 1 font-lock-variable-name-face)))

(defvar pi-def-mode-syntax-table
  (let ((st (make-syntax-table)))
    (modify-syntax-entry 35 "< b" st)
    (modify-syntax-entry 10 "> b" st)
    (modify-syntax-entry 34 "\"" st)
    st))

(defun pi-def-indent-line ()
  (interactive)
  (let ((indent-col 0)
        (is-inside-string (nth 3 (syntax-ppss))))
    (save-excursion
      (beginning-of-line)
      (cond
       (is-inside-string (setq indent-col 4))
       ((looking-at "^\\s-*\\(flag\\|arg\\|example\\|text\\)\\b")
        (setq indent-col 4))
       (t (setq indent-col 0))))
    (indent-line-to indent-col)))

(define-derived-mode pi-def-mode prog-mode "Pi-Def"
  (setq-local font-lock-defaults '(pi-def-highlights))
  (setq-local comment-start "# ")
  (setq-local indent-line-function 'pi-def-indent-line))

(add-to-list 'auto-mode-alist '("\\.def\\\'" . pi-def-mode))

(provide 'pi-def-mode)
