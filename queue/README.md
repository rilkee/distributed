# lock-free-queue

什么是 lock-free 队列？

lock-free 指的是在并发情况下，队列不通过锁来实现出队和入队操作。

一般情况下我们可以使用排他锁来实现队列的并发访问，但是锁会出现死锁的情况，同时性能也不够好（加锁和释放会耗时）。

lock-free 算法由 Maged M. Michael 和 Michael L. Scott 在1996年的论文
[《 Simple, Fast, and Practical Non-Blocking and Blocking Concurrent Queue Algorithms》](https://www.cs.rochester.edu/u/scott/papers/1996_PODC_queues.pdf)
中提出，并且给出了完整的伪代码过程：

```
structure pointer_t {ptr: pointer to node_t, count: unsigned integer}
 structure node_t {value: data type, next: pointer_t}
 structure queue_t {Head: pointer_t, Tail: pointer_t}
 
 initialize(Q: pointer to queue_t)
    node = new_node()		// Allocate a free node
    node->next.ptr = NULL	// Make it the only node in the linked list
    Q->Head.ptr = Q->Tail.ptr = node	// Both Head and Tail point to it
 
 enqueue(Q: pointer to queue_t, value: data type)
  E1:   node = new_node()	// Allocate a new node from the free list
  E2:   node->value = value	// Copy enqueued value into node
  E3:   node->next.ptr = NULL	// Set next pointer of node to NULL
  E4:   loop			// Keep trying until Enqueue is done
  E5:      tail = Q->Tail	// Read Tail.ptr and Tail.count together
  E6:      next = tail.ptr->next	// Read next ptr and count fields together
  E7:      if tail == Q->Tail	// Are tail and next consistent?
              // Was Tail pointing to the last node?
  E8:         if next.ptr == NULL
                 // Try to link node at the end of the linked list
  E9:            if CAS(&tail.ptr->next, next, <node, next.count+1>)
 E10:               break	// Enqueue is done.  Exit loop
 E11:            endif
 E12:         else		// Tail was not pointing to the last node
                 // Try to swing Tail to the next node
 E13:            CAS(&Q->Tail, tail, <next.ptr, tail.count+1>)
 E14:         endif
 E15:      endif
 E16:   endloop
        // Enqueue is done.  Try to swing Tail to the inserted node
 E17:   CAS(&Q->Tail, tail, <node, tail.count+1>)
 
 dequeue(Q: pointer to queue_t, pvalue: pointer to data type): boolean
  D1:   loop			     // Keep trying until Dequeue is done
  D2:      head = Q->Head	     // Read Head
  D3:      tail = Q->Tail	     // Read Tail
  D4:      next = head.ptr->next    // Read Head.ptr->next
  D5:      if head == Q->Head	     // Are head, tail, and next consistent?
  D6:         if head.ptr == tail.ptr // Is queue empty or Tail falling behind?
  D7:            if next.ptr == NULL  // Is queue empty?
  D8:               return FALSE      // Queue is empty, couldn't dequeue
  D9:            endif
                 // Tail is falling behind.  Try to advance it
 D10:            CAS(&Q->Tail, tail, <next.ptr, tail.count+1>)
 D11:         else		     // No need to deal with Tail
                 // Read value before CAS
                 // Otherwise, another dequeue might free the next node
 D12:            *pvalue = next.ptr->value
                 // Try to swing Head to the next node
 D13:            if CAS(&Q->Head, head, <next.ptr, head.count+1>)
 D14:               break             // Dequeue is done.  Exit loop
 D15:            endif
 D16:         endif
 D17:      endif
 D18:   endloop
 D19:   free(head.ptr)		     // It is safe now to free the old node
 D20:   return TRUE                   // Queue was not empty, dequeue succeeded
```

伪码中 CAS 指的是 “compare and swap”，是现代CPU的指令集之一，通过原子操作来更改内存中的值。

go语言可以通过atomic包来实现：
```
atomic.CompareAndSwapPointer(addr *unsafe.Pointer, old, new unsafe.Pointer) (swapped bool)
```

CAS通过比较addr位置的值和old的值，如果相等将addr位置的值更新为new，否则不更新。

### benchmark

![性能比较](https://shiniao.fun/images/20200923162432.png)




