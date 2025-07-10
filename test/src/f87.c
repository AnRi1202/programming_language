struct Node {
  long value;
  struct Node* next;
};

long f(long x) {
  struct Node n1;
  struct Node n2;
  struct Node* ptr;
  
  n1.value = x;
  n2.value = x * 2;
  n1.next = &n2;
  n2.next = 0;
  
  ptr = &n1;
  return ptr->value + ptr->next->value;
} 