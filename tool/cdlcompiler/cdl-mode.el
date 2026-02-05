;;; cdl-mode.el --- Major mode for CLI definition files (.cdl)

(defvar cdl-highlights
  '(("^#.*" . font-lock-comment-face)
    ("\\b\\(global\\|cmd\\|topic\\)\\b" . font-lock-keyword-face)
    ("\\b\\(param\\|flag\\|arg\\|example\\|text\\)\\b" . font-lock-builtin-face)
    ("\\b\\(bool\\|string\\)\\b" . font-lock-type-face)
    ("cmd\\s-+\\([a-zA-Z0-9_/ ]+\\)\)" 1 font-lock-function-name-face)
    ("\\(?:flag\\|arg\\)\\s-+\\([a-zA-Z0-9_-]+\\)" 1 font-lock-variable-name-face)))

(defvar cdl-mode-syntax-table
  (let ((st (make-syntax-table)))
    (modify-syntax-entry 35 "< b" st)
    (modify-syntax-entry 10 "> b" st)
    (modify-syntax-entry 34 "\"" st)
    st))

(defun cdl-indent-line ()
  (interactive)
  (let ((indent-col 0)
        (is-inside-string (nth 3 (syntax-ppss))))
    (save-excursion
      (beginning-of-line)
      (cond
       (is-inside-string (setq indent-col 4))
       ((looking-at "^\\s-*\\(param\\|flag\\|arg\\|example\\|text\\)\\b")
        (setq indent-col 4))
       (t (setq indent-col 0))))
    (indent-line-to indent-col)))

(define-derived-mode cdl-mode prog-mode "Cdl"
  (setq-local font-lock-defaults '(cdl-highlights))
  (setq-local comment-start "# ")
  (setq-local indent-line-function 'cdl-indent-line))

(add-to-list 'auto-mode-alist '("\\.cdl\\\'" . cdl-mode))

(provide 'cdl-mode)
