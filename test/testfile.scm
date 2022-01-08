(define (accumulate op term init a next b)  
  (define (loop i)
      (if (<= i b)
          (op (term i) (loop (next i)) )
          init
  ))
  (loop a)
)


(define (1+ num) (+ num 1))


(define (numLen num system)
  (define (loop num len)
    (if (= num 0)
        len
        (loop (quotient num system) (+ len 1))
    )
  )

  (loop num 0)
)


(define (numLen2 num) (numLen num 2))


(define (to-binary num)
  (define len (numLen2 num))
  
  (define (loop num result cnt)
    (if (= num 0)
        result
        (loop (quotient num 2) (+ result (* (remainder num 2) (expt 10 cnt))) (+ cnt 1))
    )
  )

  (loop num 0 0)
)


(define (set-contains? set elem)
  (= (remainder (quotient set (expt 2 elem)) 2) 1)
)


(define (set-add set elem)
  (if (set-contains? set elem)
      set
      (+ set (expt 2 elem))
  )
)


(define (set-remove set elem)
  (if (set-contains? set elem)
      (- set (expt 2 elem))
      set
  )
)


(define (set-empty? set)
  (= set 0)
)


(define (set-size set)
  (define (loop set result cnt)
    (if (set-empty? set)
        result
        (if (set-contains? set cnt)
            (loop (set-remove set cnt) (+ result 1) (+ cnt 1))
            (loop set result (+ cnt 1))
        )
    )
  )

  (loop set 0 0)
)


(define (set-intersect s1 s2)
  (define (term i)
    (if (and (set-contains? s1 i) (set-contains? s2 i))
        (expt 2 i)
        0
    )
  )
  
  (accumulate + term 0 0 1+ (- (min (numLen2 s1) (numLen2 s2)) 1))
)


(define (set-union s1 s2)
  (define (term i)
    (if (or (set-contains? s1 i) (set-contains? s2 i))
        (expt 2 i)
        0
    )
  )
  
  (accumulate + term 0 0 1+ (- (max (numLen2 s1) (numLen2 s2)) 1))
)


(define (set-difference s1 s2)
  (define (term i)
    (if (and (set-contains? s1 i) (not (set-contains? s2 i)))
        (expt 2 i)
        0
    )
  )
  
  (accumulate + term 0 0 1+ (- (max (numLen2 s1) (numLen2 s2)) 1))
)


(define (knapsack c n w p)
  (define (ks-rec c n w p res)
    (define (get-total-price set)
      (define (cp i)
        (if (set-contains? set i)
            (p i)
            0
            )
        )
      (accumulate + cp 0 0 1+ (- n 1))
    )

    (define s1 (if (> n 0) (ks-rec c (- n 1) w p res) 0))
    (define s2 (if (> n 0) (ks-rec (- c (w (- n 1))) (- n 1) w p (set-add res (- n 1))) 0))
  
    (if (or (= n 0) (= c 0))
        res
        (if (> (w (- n 1)) c)
            (ks-rec c (- n 1) w p res)
            (if (> (get-total-price s1) (get-total-price s2))
                s1
                (set-union res s2)
            )
        )
    )
  )

  (ks-rec c n w p 0)
)


(define (w i)
	(if (= i 0)
		10
		(if (= i 1)
			20
			(if (= i 2)
				30
				-1
			)
		)
	)
)


(define (p i)
	(if (= i 0)
		60
		(if (= i 1)
			100
			(if (= i 2)
				120
				-1
			)
		)
	)
)
