import { Route, Routes } from 'react-router-dom'
import { AppLayout } from './layout/AppLayout'
import { ClassesPage } from './pages/ClassesPage'
import { HomePage } from './pages/HomePage'
import { MyClassesPage } from './pages/MyClassesPage'

export default function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<HomePage />} />
        <Route path="classes" element={<ClassesPage />} />
        <Route path="my-classes" element={<MyClassesPage />} />
      </Route>
    </Routes>
  )
}
