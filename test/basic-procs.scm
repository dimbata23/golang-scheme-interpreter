(define (caar pair)
	(car (car pair))
)

(define (cddr pair)
	(cdr (cdr pair))
)

(define (cadr pair)
	(car (cdr pair))
)

(define (cdar pair)
	(cdr (car pair))
)

(define (not x)
	(if x #f #t)
)

(define (even? x)
	(= (remainder x 2) 0)
)

(define (odd? x)
	(not (even? x))
)

(define (atom? x)
	(not (pair? x))
)


(define (length lst)
	(define (loop lst res)
		(if (null? lst)
			res
			(loop (cdr lst) (+ res 1))
		)
	)
	
	(loop lst 0)
)


(define (append lst1 lst2)
	(if (null? lst1)
		lst2
		(cons (car lst1)
			  (append (cdr lst1) lst2)
		)
	)
)


(define (reverse lst)
	(define (loop lst result)
		(if (null? lst)
			result
			(loop (cdr lst) (cons (car lst) result))
		)
	)
	(loop lst '())
)


(define (append-iter lst1 lst2)
	(define (loop lst result)
		(if (null? lst)
			result
			(loop (cdr lst) (cons (car lst) result))
		)
	)

	(loop (reverse lst1) lst2)
)


(define (foldr proc end lst)
	(if (null? lst)
		end
		(proc (car lst) (foldr proc end (cdr lst)))
	)
)


(define (foldl proc accum lst)
	(if (null? lst)
		accum
		(foldl proc (proc accum (car lst)) (cdr lst))
	)
)


(define (map-f func lst)
	(foldr (lambda (x y) (cons (func x) y)) '() lst)
)


(define (filter-f pred lst)
	(foldr (lambda (x y) (if (pred x) (cons x y) y)) '() lst)
)


(define (filter pred? L)
	(cond
		((null? L) '())
		((pred? (car L))
			(cons (car L)
			(filter pred? (cdr L)))
		)
		(else
			(filter pred? (cdr L))
		)
	)
)


(define (flatten lst)
	(cond
		((null? lst) '())
		((pair? (car lst))
			(append (flatten (car lst))
					(flatten (cdr lst))
			)
		)
		(else
			(cons (car lst)
				  (flatten (cdr lst))
			)
		)
	)
)
