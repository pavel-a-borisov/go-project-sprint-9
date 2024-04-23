package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	var i int64 = 1 // начальное значение N(0) = 1
	for {
		select {
		case <-ctx.Done(): // прекращаем работу при поступлении сигнала об отмене контекста
			close(ch) // Закрываем канал перед выходом из функции.
			return
		case ch <- i: // записываем значени N(i) в канал
			fn(i) // вызываем функцию fn
			i++   // увеличиваем значение N(i) = N(i-1) + 1
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	// 2. Функция Worker
	for v := range in {
		out <- v                         // отправляем результат в канал out
		time.Sleep(1 * time.Millisecond) // делаем паузу 1 миллисекунду
	}
	close(out) // закрываем канал по окончании работы функции
}

func main() {
	chIn := make(chan int64)

	// 3. Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), time.Second) // создаем контекст, который отменяется через 1 секунду.
	defer cancel()                                                        // Отложенный вызов функции cancel для корректного освобождения ресурсов контекста.

	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		// Код ниже изменене с использованием атомарных операций
		// inputSum += i
		// inputCount++
		atomic.AddInt64(&inputSum, i)   // Используем атомарное сложение для подсчета суммы
		atomic.AddInt64(&inputCount, 1) // Используем атомарное сложение для счетчика чисел
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut) // слайс для подсчета количества чисел, которые прошли через каждый канал outs
	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	// 4. Собираем числа из каналов outs
	for i, out := range outs {
		wg.Add(1) // инкрементируем счётчик перед запуском горутины
		go func(in <-chan int64, index int) {
			defer wg.Done() // уменьшаем счётчик, когда горутина завершает работу
			for v := range in {
				atomic.AddInt64(&amounts[index], 1) // Атомарное увеличение счетчика обработанных чисел для данного канала.
				chOut <- v                          // Отправляем число в выходной канал.
			}
		}(out, i)
	}

	go func() {
		// ждём завершения работы всех горутин для outs
		wg.Wait()
		// закрываем результирующий канал
		close(chOut)
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала
	for n := range chOut {
		atomic.AddInt64(&count, 1) // Атомарное увеличение счетчика чисел.
		atomic.AddInt64(&sum, n)   // Атомарное добавление значения числа к общей сумме.
	}

	// Код ниже изменене с использованием атомарных операций
	//fmt.Println("Количество чисел", inputCount, count)
	//fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Количество чисел", atomic.LoadInt64(&inputCount), count)
	fmt.Println("Сумма чисел", atomic.LoadInt64(&inputSum), sum)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	// Код ниже изменене с использованием атомарных операций
	//if inputSum != sum {
	//	log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	//}
	if atomic.LoadInt64(&inputSum) != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %dn", atomic.LoadInt64(&inputSum), sum)
	}

	//if inputCount != count {
	//	log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	//}
	if atomic.LoadInt64(&inputCount) != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %dn", atomic.LoadInt64(&inputCount), count)
	}

	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
