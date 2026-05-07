export type Class = {
  id: string
  title: string
  description?: string
  capacity: number
  status: string
  enrollment_opens_at?: string
  enrollment_closes_at?: string
  created_at: string
  active_enrollments: number
}

export type MyClass = Class & {
  enrollment_status: string
  enrolled_at: string
}

export type ApiErrorBody = {
  error: string
}
