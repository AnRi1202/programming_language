typedef long* LongPtr;
typedef LongPtr* LongPtrPtr;

long f(long x) {
  long y = x + 1;
  LongPtr ptr1 = &y;
  LongPtrPtr ptr2 = &ptr1;
  
  return **ptr2;
} 