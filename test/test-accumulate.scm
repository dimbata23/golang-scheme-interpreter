(define (id x) x)


(define (even+ i result)
	(if (even? i) (+ i result) result)
)


(define (accumulate op term init a next b)  
  (define (loop i)
      (if (<= i b)
          (op (term i) (loop (next i)) )
          init
  ))
  (loop a)
)


(define (sum-even a b)
	(accumulate even+ id 0 a 1+ b)
)
